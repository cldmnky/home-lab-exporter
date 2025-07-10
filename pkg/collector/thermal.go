package collector

import (
	"encoding/json"
	"os/exec"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ThermalData struct {
	Temperatures []struct {
		Name           string  `json:"Name"`
		ReadingCelsius float64 `json:"ReadingCelsius"`
		Status         struct {
			Health string `json:"Health"`
		} `json:"Status"`
	} `json:"Temperatures"`
	Fans []struct {
		Name    string  `json:"Name"`
		Reading float64 `json:"Reading"`
		Status  struct {
			Health string `json:"Health"`
		} `json:"Status"`
	} `json:"Fans"`
}

type ThermalCollector struct {
	mutex       sync.Mutex
	cache       ThermalData
	target      string
	username    string
	password    string
	temperature *prometheus.GaugeVec
	fanSpeed    *prometheus.GaugeVec
}

func NewThermalCollector(target, username, password string) *ThermalCollector {
	collector := &ThermalCollector{
		target:   target,
		username: username,
		password: password,
		temperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "redfish_temperature_celsius",
				Help: "Temperature readings from Redfish",
			},
			[]string{"sensor", "name", "target", "health"},
		),
		fanSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "redfish_fan_speed_rpm",
				Help: "Fan speeds from Redfish",
			},
			[]string{"fan", "name", "target", "health"},
		),
	}

	go collector.run()
	return collector
}

func (c *ThermalCollector) Describe(ch chan<- *prometheus.Desc) {
	c.temperature.Describe(ch)
	c.fanSpeed.Describe(ch)
}

func (c *ThermalCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.temperature.Reset()
	for _, temp := range c.cache.Temperatures {
		c.temperature.WithLabelValues(temp.Name, "temperature", c.target, temp.Status.Health).Set(temp.ReadingCelsius)
	}

	c.fanSpeed.Reset()
	for _, fan := range c.cache.Fans {
		c.fanSpeed.WithLabelValues(fan.Name, "fan", c.target, fan.Status.Health).Set(fan.Reading)
	}

	c.temperature.Collect(ch)
	c.fanSpeed.Collect(ch)
}

func (c *ThermalCollector) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		c.fetch()
		<-ticker.C
	}
}

func (c *ThermalCollector) fetch() {
	cmd := exec.Command("redfishtool", "-r", c.target, "-u", c.username, "-p", c.password, "Chassis", "-I", "Baseboard", "Thermal")
	output, err := cmd.Output()
	if err != nil {
		return // You may want to log the error
	}

	var data ThermalData
	if err := json.Unmarshal(output, &data); err != nil {
		return
	}

	c.mutex.Lock()
	c.cache = data
	c.mutex.Unlock()
}
