package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
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
	// Use gofish to fetch thermal data
	cfg := gofish.ClientConfig{
		Endpoint:              "https://" + c.target,
		Username:              c.username,
		Password:              c.password,
		Insecure:              true, // Set to false if you want to verify SSL certificates
		MaxConcurrentRequests: 3,
		ReuseConnections:      true,
	}
	client, err := gofish.Connect(cfg)
	if err != nil {
		log.Printf("Error connecting to Redfish target: %v", err)
		return
	}
	defer client.Logout()
	service := client.Service

	chass, err := service.Chassis()
	if err != nil {
		log.Printf("Error fetching chassis: %v", err)
		return
	}
	log.Println("--------- Chassis count:", len(chass), "---------")
	for _, ch := range chass {
		if therm, err := ch.Thermal(); err != nil || therm == nil {
			continue
		}
		log.Printf("Chassis: %s, Description: %s", ch.Name, ch.Description)
		therm, err := ch.Thermal()
		if err != nil {
			log.Printf("Error fetching thermal data for chassis %s: %v", ch.Name, err)
			continue
		}
		// unmarshal therm.Entries to ThermalData using mapstruct
		data := ThermalData{
			Temperatures: make([]struct {
				Name           string  `json:"Name"`
				ReadingCelsius float64 `json:"ReadingCelsius"`
				Status         struct {
					Health string `json:"Health"`
				} `json:"Status"`
			}, 0),
			Fans: make([]struct {
				Name    string  `json:"Name"`
				Reading float64 `json:"Reading"`
				Status  struct {
					Health string `json:"Health"`
				} `json:"Status"`
			}, 0),
		}
		for _, temp := range therm.Temperatures {
			data.Temperatures = append(data.Temperatures, struct {
				Name           string  `json:"Name"`
				ReadingCelsius float64 `json:"ReadingCelsius"`
				Status         struct {
					Health string `json:"Health"`
				} `json:"Status"`
			}{
				Name:           temp.Name,
				ReadingCelsius: float64(temp.ReadingCelsius),
				Status: struct {
					Health string `json:"Health"`
				}{Health: string(temp.Status.Health)},
			})
		}
		for _, fan := range therm.Fans {
			data.Fans = append(data.Fans, struct {
				Name    string  `json:"Name"`
				Reading float64 `json:"Reading"`
				Status  struct {
					Health string `json:"Health"`
				} `json:"Status"`
			}{
				Name:    fan.Name,
				Reading: float64(fan.Reading),
				Status: struct {
					Health string `json:"Health"`
				}{Health: string(fan.Status.Health)},
			})
		}
		c.mutex.Lock()
		c.cache = data
		c.mutex.Unlock()
	}
}
