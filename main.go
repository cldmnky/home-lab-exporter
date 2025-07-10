package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/unpoller/unifi/v5"

	"github.com/cldmnky/home-lab-exporter/pkg/collector"
)

type Config struct {
	ListenAddr    string
	RedfishTarget string
	RedfishUser   string
	RedfishPass   string
	UniFiURL      string
	UniFiUser     string
	UniFiPass     string
}

func initConfig() *Config {
	pflag.String("listen", ":9100", "HTTP listen address")
	pflag.String("redfish.target", "", "Redfish target address")
	pflag.String("redfish.user", "", "Redfish username")
	pflag.String("redfish.password", "", "Redfish password")
	pflag.String("unifi.url", "", "UniFi controller URL")
	pflag.String("unifi.user", "", "UniFi controller username")
	pflag.String("unifi.pass", "", "UniFi controller password")
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.BindPFlags(pflag.CommandLine)

	return &Config{
		ListenAddr:    viper.GetString("listen"),
		RedfishTarget: viper.GetString("redfish.target"),
		RedfishUser:   viper.GetString("redfish.user"),
		RedfishPass:   viper.GetString("redfish.password"),
		UniFiURL:      viper.GetString("unifi.url"),
		UniFiUser:     viper.GetString("unifi.user"),
		UniFiPass:     viper.GetString("unifi.password"),
	}
}

func main() {
	cfg := initConfig()

	// Optional: Validate config
	if cfg.RedfishTarget == "" || cfg.UniFiURL == "" {
		log.Fatalln("At least one of Redfish and UniFi config must be provided")
	}

	c := unifi.Config{
		User:     cfg.UniFiUser,
		Pass:     cfg.UniFiPass,
		URL:      cfg.UniFiURL,
		ErrorLog: log.Printf,
		//DebugLog: log.Printf,
	}
	client, err := unifi.NewUnifi(&c)
	if err != nil {
		log.Fatalln("Error creating UniFi client:", err)
	}

	thermalCollector := collector.NewThermalCollector(cfg.RedfishTarget, cfg.RedfishUser, cfg.RedfishPass)
	unifiCollector := collector.NewUniFiCollectorWithClient(client)
	prometheus.MustRegister(thermalCollector, unifiCollector)

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting exporter on ", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, nil))
}
