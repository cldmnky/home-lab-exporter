package collector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	unifi "github.com/unpoller/unifi/v5"
)

type mockClient struct {
	unifi.Unifi
	loggedIn bool
	Sites    []*unifi.Site
	Clients  []*unifi.Client
	Devices  *unifi.Devices
	Err      error
}

func (m *mockClient) Login() error {
	m.loggedIn = true
	return nil
}

func (m *mockClient) GetSites() ([]*unifi.Site, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Sites, nil
}

func (m *mockClient) GetClients(_ []*unifi.Site) ([]*unifi.Client, error) {
	return m.Clients, nil
}

func (m *mockClient) GetDevices(_ []*unifi.Site) (*unifi.Devices, error) {
	return m.Devices, nil
}

func TestCollectorCollect(t *testing.T) {
	mc := &mockClient{
		Sites:   []*unifi.Site{{Name: "default", ID: "site-id"}},
		Clients: []*unifi.Client{{Name: "client1", IP: "192.168.1.100", ApMac: "aa:bb:cc", Rssi: *unifi.NewFlexInt(-50), TxBytes: *unifi.NewFlexInt(1000), RxBytes: *unifi.NewFlexInt(2000)}},
		Devices: &unifi.Devices{
			UAPs: []*unifi.UAP{{
				Name:        "uap-1",
				IP:          "192.168.1.2",
				NumSta:      *unifi.NewFlexInt(3),
				SystemStats: unifi.SystemStats{CPU: *unifi.NewFlexInt(10), Mem: *unifi.NewFlexInt(20)},
				Uplink: struct {
					FullDuplex       unifi.FlexBool `json:"full_duplex"`
					IP               string         `fake:"{ipv4address}" json:"ip"`
					Mac              string         `fake:"{macaddress}"  json:"mac"`
					MaxVlan          int            `json:"max_vlan"`
					Name             string         `json:"name"`
					Netmask          string         `json:"netmask"`
					NumPort          int            `fake:"{port}"        json:"num_port"`
					RxBytes          unifi.FlexInt  `json:"rx_bytes"`
					RxDropped        unifi.FlexInt  `json:"rx_dropped"`
					RxErrors         unifi.FlexInt  `json:"rx_errors"`
					RxMulticast      unifi.FlexInt  `json:"rx_multicast"`
					RxPackets        unifi.FlexInt  `json:"rx_packets"`
					Speed            unifi.FlexInt  `json:"speed"`
					TxBytes          unifi.FlexInt  `json:"tx_bytes"`
					TxDropped        unifi.FlexInt  `json:"tx_dropped"`
					TxErrors         unifi.FlexInt  `json:"tx_errors"`
					TxPackets        unifi.FlexInt  `json:"tx_packets"`
					Up               unifi.FlexBool `json:"up"`
					MaxSpeed         unifi.FlexInt  `json:"max_speed"`
					Type             string         `json:"type"`
					TxBytesR         unifi.FlexInt  `json:"tx_bytes-r"`
					RxBytesR         unifi.FlexInt  `json:"rx_bytes-r"`
					UplinkMac        string         `fake:"{macaddress}"  json:"uplink_mac"`
					UplinkRemotePort int            `fake:"{port}"        json:"uplink_remote_port"`
				}{
					RxBytes: *unifi.NewFlexInt(1000),
					TxBytes: *unifi.NewFlexInt(2000),
				},
			}},
		},
	}

	col := NewUniFiCollectorWithClient(mc)

	err := col.fetch()
	assert.NoError(t, err)

	registry := prometheus.NewRegistry()
	registry.MustRegister(col)

	// Wait to ensure metrics are populated
	time.Sleep(1 * time.Second)

	// Test if a specific metric exists and has the expected value
	count := testutil.CollectAndCount(col)
	t.Logf("Collected %d metrics", count)
	assert.Greater(t, count, 0)

	// Check device temperature
	tempVal := testutil.ToFloat64(col.deviceTemp.WithLabelValues("uap-1", "192.168.1.2"))
	assert.Equal(t, 0.0, tempVal) // Assuming no temperature data is set in mock
	cpuVal := testutil.ToFloat64(col.deviceCPU.WithLabelValues("uap-1", "192.168.1.2"))
	assert.Equal(t, 10.0, cpuVal)
	memVal := testutil.ToFloat64(col.deviceMem.WithLabelValues("uap-1", "192.168.1.2"))
	assert.Equal(t, 20.0, memVal)
}
