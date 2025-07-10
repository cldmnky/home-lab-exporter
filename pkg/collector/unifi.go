package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/unpoller/unifi/v5"
)

type UnifiData struct {
	Sites   []unifi.Site
	Devices UnifiDevices
	Clients []unifi.Client
}

type UnifiDevice interface {
	Name() string
	Site() string
	IP() string
	HasTemperature() bool
	Temperature() float64
	Model() string
	Type() string
	CPUUsage() float64
	MEMUsage() float64
}

type udmAdapter struct{ *unifi.UDM }

func (d udmAdapter) Name() string         { return d.UDM.Name }
func (d udmAdapter) Site() string         { return d.UDM.SiteName }
func (d udmAdapter) IP() string           { return d.UDM.IP }
func (d udmAdapter) HasTemperature() bool { return d.UDM.HasTemperature.Val }
func (d udmAdapter) Temperature() float64 {
	if len(d.UDM.Temperatures) > 0 {
		return float64(d.UDM.Temperatures[0].Value)
	}
	return 0
}
func (d udmAdapter) Model() string { return d.UDM.Model }
func (d udmAdapter) Type() string  { return "UDM" }
func (d udmAdapter) CPUUsage() float64 {
	if d.UDM.SystemStats.CPU.Val < 0 {
		return 0
	}
	return d.UDM.SystemStats.CPU.Val
}
func (d udmAdapter) MEMUsage() float64 {
	if d.UDM.SystemStats.Mem.Val < 0 {
		return 0
	}
	return d.UDM.SystemStats.Mem.Val
}

type usgAdapter struct{ *unifi.USG }

func (d usgAdapter) Name() string         { return d.USG.Name }
func (d usgAdapter) Site() string         { return d.USG.SiteName }
func (d usgAdapter) IP() string           { return d.USG.IP }
func (d usgAdapter) HasTemperature() bool { return false }
func (d usgAdapter) Temperature() float64 { return 0 }
func (d usgAdapter) Model() string        { return d.USG.Model }
func (d usgAdapter) Type() string         { return "USG" }
func (d usgAdapter) CPUUsage() float64 {
	if d.USG.SystemStats.CPU.Val < 0 {
		return 0
	}
	return d.USG.SystemStats.CPU.Val
}
func (d usgAdapter) MEMUsage() float64 {
	if d.USG.SystemStats.Mem.Val < 0 {
		return 0
	}
	return d.USG.SystemStats.Mem.Val
}

type uswAdapter struct{ *unifi.USW }

func (d uswAdapter) Name() string         { return d.USW.Name }
func (d uswAdapter) Site() string         { return d.USW.SiteName }
func (d uswAdapter) IP() string           { return d.USW.IP }
func (d uswAdapter) HasTemperature() bool { return d.USW.HasTemperature.Val }
func (d uswAdapter) Temperature() float64 { return d.USW.GeneralTemperature.Val }
func (d uswAdapter) Model() string        { return d.USW.Model }
func (d uswAdapter) Type() string         { return "USW" }
func (d uswAdapter) CPUUsage() float64 {
	if d.USW.SystemStats.CPU.Val < 0 {
		return 0
	}
	return d.USW.SystemStats.CPU.Val
}
func (d uswAdapter) MEMUsage() float64 {
	if d.USW.SystemStats.Mem.Val < 0 {
		return 0
	}
	return d.USW.SystemStats.Mem.Val
}

type uapAdapter struct{ *unifi.UAP }

func (d uapAdapter) Name() string         { return d.UAP.Name }
func (d uapAdapter) Site() string         { return d.UAP.SiteName }
func (d uapAdapter) IP() string           { return d.UAP.IP }
func (d uapAdapter) HasTemperature() bool { return false } // most UAPs don't report temperature
func (d uapAdapter) Temperature() float64 { return 0 }
func (d uapAdapter) Model() string        { return d.UAP.Model }
func (d uapAdapter) Type() string         { return "UAP" }
func (d uapAdapter) CPUUsage() float64 {
	if d.UAP.SystemStats.CPU.Val < 0 {
		return 0
	}
	return d.UAP.SystemStats.CPU.Val
}
func (d uapAdapter) MEMUsage() float64 {
	if d.UAP.SystemStats.Mem.Val < 0 {
		return 0
	}
	return d.UAP.SystemStats.Mem.Val
}

type UnifiDevices struct {
	UDMs []unifi.UDM
	USGs []unifi.USG
	USWs []unifi.USW
	UAPs []unifi.UAP
}

func (d UnifiDevices) All() []UnifiDevice {
	var all []UnifiDevice
	for _, dev := range d.UDMs {
		all = append(all, udmAdapter{&dev})
	}
	for _, dev := range d.USGs {
		all = append(all, usgAdapter{&dev})
	}
	for _, dev := range d.USWs {
		all = append(all, uswAdapter{&dev})
	}
	for _, dev := range d.UAPs {
		all = append(all, uapAdapter{&dev})
	}
	return all
}

type UniFiClient interface {
	GetSites() ([]*unifi.Site, error)
	GetClients([]*unifi.Site) ([]*unifi.Client, error)
	GetDevices([]*unifi.Site) (*unifi.Devices, error)
	Login() error
}

type UniFiCollector struct {
	client UniFiClient
	mutex  sync.Mutex
	cache  UnifiData
	// Device metrics
	deviceTemp *prometheus.GaugeVec
	deviceCPU  *prometheus.GaugeVec
	deviceMem  *prometheus.GaugeVec
	// Switch metrics for usw
	swRXPackets *prometheus.CounterVec // d.Stat.Sw.RxPackets
	swRXBytes   *prometheus.CounterVec // d.Stat.Sw.RxBytes
	swRXErrors  *prometheus.CounterVec // d.Stat.Sw.RxErrors
	swRXDropped *prometheus.CounterVec // d.Stat.Sw.RxDropped
	swTXPackets *prometheus.CounterVec // d.Stat.Sw.TxPackets
	swTXBytes   *prometheus.CounterVec // d.Stat.Sw.TxBytes
	swTXErrors  *prometheus.CounterVec // d.Stat.Sw.TxErrors
	swTXDropped *prometheus.CounterVec // d.Stat.Sw.TxDropped
	swBytes     *prometheus.CounterVec // d.Stat.Sw.Bytes
	// Port metrics for usw and udm
	pRXPackets *prometheus.CounterVec // d.PortTable[i].RxPackets
	pRXBytes   *prometheus.CounterVec // d.PortTable[i].RxBytes
	pRXErrors  *prometheus.CounterVec // d.PortTable[i].RxErrors
	pRXDropped *prometheus.CounterVec // d.PortTable[i].RxDropped
	pSpeed     *prometheus.GaugeVec   // d.PortTable[i].Speed
	pTXPackets *prometheus.CounterVec // d.PortTable[i].TxPackets
	pTXBytes   *prometheus.CounterVec // d.PortTable[i].TxBytes
	pTXErrors  *prometheus.CounterVec // d.PortTable[i].TxErrors
	pTXDropped *prometheus.CounterVec // d.PortTable[i].TxDropped
	pSFPTemp   *prometheus.GaugeVec   // if SFPFound.Val -> d.PortTable[i].SFPTemp
	// Removed for now
	/*
		portTx        *prometheus.GaugeVec
		uplinkRxBytes *prometheus.GaugeVec
		uplinkTxBytes *prometheus.GaugeVec
		clientRssi    *prometheus.GaugeVec
		apClients     *prometheus.GaugeVec
		radioRxBytes  *prometheus.GaugeVec
		radioTxBytes  *prometheus.GaugeVec
	*/
}

func NewUniFiCollectorWithClient(client UniFiClient) *UniFiCollector {
	labels := []string{"type", "site", "source", "name"}
	portLabels := []string{"type", "site", "source", "name", "port", "port_number", "up", "uplink"}
	col := &UniFiCollector{
		client:     client,
		deviceTemp: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "unifi_device_temperature_celsius", Help: "Device temp (°C)"}, labels),
		deviceCPU:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "unifi_device_cpu_pct", Help: "Device CPU (%)"}, labels),
		deviceMem:  prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "unifi_device_mem_pct", Help: "Device memory (%)"}, labels),
		// Switch metrics for usw
		swRXPackets: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_rx_packets_total", Help: "Switch RX packets"}, labels),
		swRXBytes:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_rx_bytes_total", Help: "Switch RX bytes"}, labels),
		swRXErrors:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_rx_errors_total", Help: "Switch RX errors"}, labels),
		swRXDropped: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_rx_dropped_total", Help: "Switch RX dropped"}, labels),
		swTXPackets: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_tx_packets_total", Help: "Switch TX packets"}, labels),
		swTXBytes:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_tx_bytes_total", Help: "Switch TX bytes"}, labels),
		swTXErrors:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_tx_errors_total", Help: "Switch TX errors"}, labels),
		swTXDropped: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_tx_dropped_total", Help: "Switch TX dropped"}, labels),
		swBytes:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_switch_bytes_total", Help: "Switch total bytes"}, labels),

		// Port metrics for usw and udm
		pRXPackets: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_rx_packets_total", Help: "Port RX packets"}, portLabels),
		pRXBytes:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_rx_bytes_total", Help: "Port RX bytes"}, portLabels),
		pRXErrors:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_rx_errors_total", Help: "Port RX errors"}, portLabels),
		pRXDropped: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_rx_dropped_total", Help: "Port RX dropped"}, portLabels),
		pSpeed:     prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "unifi_port_speed_bps", Help: "Port speed (bps)"}, portLabels),
		pTXPackets: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_tx_packets_total", Help: "Port TX packets"}, portLabels),
		pTXBytes:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_tx_bytes_total", Help: "Port TX bytes"}, portLabels),
		pTXErrors:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_tx_errors_total", Help: "Port TX errors"}, portLabels),
		pTXDropped: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "unifi_port_tx_dropped_total", Help: "Port TX dropped"}, portLabels),
		pSFPTemp:   prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "unifi_port_sfp_temperature_celsius", Help: "Port SFP temperature (°C)"}, portLabels),
	}

	go col.run()

	return col
}

func (c *UniFiCollector) Describe(ch chan<- *prometheus.Desc) {
	c.deviceTemp.Describe(ch)
	c.deviceCPU.Describe(ch)
	c.deviceMem.Describe(ch)
	// Switch metrics
	c.swRXPackets.Describe(ch)
	c.swRXBytes.Describe(ch)
	c.swRXErrors.Describe(ch)
	c.swRXDropped.Describe(ch)
	c.swTXPackets.Describe(ch)
	c.swTXBytes.Describe(ch)
	c.swTXErrors.Describe(ch)
	c.swTXDropped.Describe(ch)
	c.swBytes.Describe(ch)
	// Port metrics
	c.pRXPackets.Describe(ch)
	c.pRXBytes.Describe(ch)
	c.pRXErrors.Describe(ch)
	c.pRXDropped.Describe(ch)
	c.pSpeed.Describe(ch)
	c.pTXPackets.Describe(ch)
	c.pTXBytes.Describe(ch)
	c.pTXErrors.Describe(ch)
	c.pTXDropped.Describe(ch)
	c.pSFPTemp.Describe(ch)
}
func (c *UniFiCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Reset all metrics before collecting new data
	resetAll(c)

	for _, dev := range c.cache.Devices.All() {
		labelValues := []string{dev.Type(), dev.Site(), dev.IP(), dev.Name()}
		c.deviceTemp.WithLabelValues(dev.Model(), dev.Site(), dev.IP(), dev.Name()).Set(dev.Temperature())
		c.deviceCPU.WithLabelValues(dev.Model(), dev.Site(), dev.IP(), dev.Name()).Set(dev.CPUUsage())
		c.deviceMem.WithLabelValues(dev.Model(), dev.Site(), dev.IP(), dev.Name()).Set(dev.MEMUsage())

		// Switch metrics for USW
		if usw, ok := dev.(uswAdapter); ok {
			stat := usw.USW.Stat.Sw
			c.swRXPackets.WithLabelValues(labelValues...).Add(float64(stat.RxPackets.Val))
			c.swRXBytes.WithLabelValues(labelValues...).Add(float64(stat.RxBytes.Val))
			c.swRXErrors.WithLabelValues(labelValues...).Add(float64(stat.RxErrors.Val))
			c.swRXDropped.WithLabelValues(labelValues...).Add(float64(stat.RxDropped.Val))
			c.swTXPackets.WithLabelValues(labelValues...).Add(float64(stat.TxPackets.Val))
			c.swTXBytes.WithLabelValues(labelValues...).Add(float64(stat.TxBytes.Val))
			c.swTXErrors.WithLabelValues(labelValues...).Add(float64(stat.TxErrors.Val))
			c.swTXDropped.WithLabelValues(labelValues...).Add(float64(stat.TxDropped.Val))
			c.swBytes.WithLabelValues(labelValues...).Add(float64(stat.Bytes.Val))

			// Port metrics
			for _, port := range usw.USW.PortTable {
				portLabels := append(labelValues, port.Name, port.PortIdx.String(), port.Up.String(), port.IsUplink.String())
				c.pRXPackets.WithLabelValues(portLabels...).Add(float64(port.RxPackets.Val))
				c.pRXBytes.WithLabelValues(portLabels...).Add(float64(port.RxBytes.Val))
				c.pRXErrors.WithLabelValues(portLabels...).Add(float64(port.RxErrors.Val))
				c.pRXDropped.WithLabelValues(portLabels...).Add(float64(port.RxDropped.Val))
				c.pSpeed.WithLabelValues(portLabels...).Set(float64(port.Speed.Val))
				c.pTXPackets.WithLabelValues(portLabels...).Add(float64(port.TxPackets.Val))
				c.pTXBytes.WithLabelValues(portLabels...).Add(float64(port.TxBytes.Val))
				c.pTXErrors.WithLabelValues(portLabels...).Add(float64(port.TxErrors.Val))
				c.pTXDropped.WithLabelValues(portLabels...).Add(float64(port.TxDropped.Val))
				if port.SFPFound.Val {
					c.pSFPTemp.WithLabelValues(portLabels...).Set(float64(port.SFPTemperature.Val))
				}
			}
		}
		// Port metrics for UDM
		if udm, ok := dev.(udmAdapter); ok {
			for _, port := range udm.UDM.PortTable {
				portLabels := append(labelValues, port.Name, port.PortIdx.String(), port.Up.String(), port.IsUplink.String())
				c.pRXPackets.WithLabelValues(portLabels...).Add(float64(port.RxPackets.Val))
				c.pRXBytes.WithLabelValues(portLabels...).Add(float64(port.RxBytes.Val))
				c.pRXErrors.WithLabelValues(portLabels...).Add(float64(port.RxErrors.Val))
				c.pRXDropped.WithLabelValues(portLabels...).Add(float64(port.RxDropped.Val))
				c.pSpeed.WithLabelValues(portLabels...).Set(float64(port.Speed.Val))
				c.pTXPackets.WithLabelValues(portLabels...).Add(float64(port.TxPackets.Val))
				c.pTXBytes.WithLabelValues(portLabels...).Add(float64(port.TxBytes.Val))
				c.pTXErrors.WithLabelValues(portLabels...).Add(float64(port.TxErrors.Val))
				c.pTXDropped.WithLabelValues(portLabels...).Add(float64(port.TxDropped.Val))
				if port.SFPFound.Val {
					c.pSFPTemp.WithLabelValues(portLabels...).Set(float64(port.SFPTemperature.Val))
				}
			}
		}
	}
	c.deviceTemp.Collect(ch)
	c.deviceCPU.Collect(ch)
	c.deviceMem.Collect(ch)
	c.swRXPackets.Collect(ch)
	c.swRXBytes.Collect(ch)
	c.swRXErrors.Collect(ch)
	c.swRXDropped.Collect(ch)
	c.swTXPackets.Collect(ch)
	c.swTXBytes.Collect(ch)
	c.swTXErrors.Collect(ch)
	c.swTXDropped.Collect(ch)
	c.swBytes.Collect(ch)
	c.pRXPackets.Collect(ch)
	c.pRXBytes.Collect(ch)
	c.pRXErrors.Collect(ch)
	c.pRXDropped.Collect(ch)
	c.pSpeed.Collect(ch)
	c.pTXPackets.Collect(ch)
	c.pTXBytes.Collect(ch)
	c.pTXErrors.Collect(ch)
	c.pTXDropped.Collect(ch)
	c.pSFPTemp.Collect(ch)
}

func resetAll(c *UniFiCollector) {
	c.deviceTemp.Reset()
	c.deviceCPU.Reset()
	c.deviceMem.Reset()
	c.swRXPackets.Reset()
	c.swRXBytes.Reset()
	c.swRXErrors.Reset()
	c.swRXDropped.Reset()
	c.swTXPackets.Reset()
	c.swTXBytes.Reset()
	c.swTXErrors.Reset()
	c.swTXDropped.Reset()
	c.swBytes.Reset()
	c.pRXPackets.Reset()
	c.pRXBytes.Reset()
	c.pRXErrors.Reset()
	c.pRXDropped.Reset()
	c.pSpeed.Reset()
	c.pTXPackets.Reset()
	c.pTXBytes.Reset()
	c.pTXErrors.Reset()
	c.pTXDropped.Reset()
	c.pSFPTemp.Reset()
}

func (c *UniFiCollector) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		if err := c.fetch(); err != nil {
			log.Println("Error fetching UniFi data:", err)
		}
		<-ticker.C
	}
}

// fetchData fetches data from the UniFi controller
func (c *UniFiCollector) fetch() error {
	if _, err := c.client.GetSites(); err != nil {
		if err := c.client.Login(); err != nil {
			log.Println("UniFi login error:", err)
			return err
		}
	}

	sites, _ := c.client.GetSites()
	clients, _ := c.client.GetClients(sites)
	devices, _ := c.client.GetDevices(sites)

	var siteVals []unifi.Site
	for _, s := range sites {
		if s != nil {
			siteVals = append(siteVals, *s)
		}
	}
	var clientVals []unifi.Client
	for _, cp := range clients {
		if cp != nil {
			clientVals = append(clientVals, *cp)
		}
	}
	var udms []unifi.UDM
	var usgs []unifi.USG
	var usw []unifi.USW
	var uaps []unifi.UAP
	for _, d := range devices.UDMs {
		if d == nil {
			continue
		}
		udms = append(udms, *d)
	}
	for _, d := range devices.USGs {
		if d == nil {
			continue
		}
		usgs = append(usgs, *d)
	}
	for _, d := range devices.USWs {
		if d == nil {
			continue
		}
		usw = append(usw, *d)
	}
	for _, d := range devices.UAPs {
		if d == nil {
			continue
		}
		uaps = append(uaps, *d)
	}
	// create the cache
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// Reset the cache before updating
	c.cache = UnifiData{
		Sites: siteVals,
		Devices: UnifiDevices{
			UDMs: udms,
			USGs: usgs,
			USWs: usw,
			UAPs: uaps,
		},
		Clients: clientVals,
	}
	return nil
}
