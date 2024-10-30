package main

import (
	"log"
	"net/http"
	"os"

	"github.com/FonovAD/PacketSleuth/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := NewConfig()
	cfg.ParseConfigWithDefaults()

	cPacket := metrics.NewPacketMonitor()
	m := metrics.NewMonitor(cPacket.Listen(), cfg.influxURL, cfg.influxUser, cfg.influxPass, cfg.influxOrg, cfg.influxBucket)
	go m.Start()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":2112", nil))
}

type Config struct {
	influxURL    string
	influxOrg    string
	influxBucket string
	influxUser   string
	influxPass   string
}

const (
	defaultInfluxURL    = "http://localhost:8086"
	defaultInfluxOrg    = "myorg"
	defaultInfluxBucket = "mybucket"
	defaultInfluxUser   = "admin"
	defaultInfluxPass   = "password"
)

func NewConfig() *Config {
	return &Config{}
}
func (c *Config) ParseConfigWithDefaults() {
	influxURL := os.Getenv("INFLUXDB_HOST")
	influxOrg := os.Getenv("INFLUXDB_ORG")
	influxBucket := os.Getenv("INFLUXDB_BUCKET")
	influxUser := os.Getenv("INFLUXDB_USER")
	influxPass := os.Getenv("INFLUXDB_PASSWORD")
	if influxURL == "" {
		c.influxURL = defaultInfluxURL
	}
	if influxOrg == "" {
		c.influxOrg = defaultInfluxOrg
	}
	if influxBucket == "" {
		c.influxBucket = defaultInfluxBucket
	}
	if influxUser == "" {
		c.influxUser = defaultInfluxUser
	}
	if influxPass == "" {
		c.influxPass = defaultInfluxPass
	}
}
