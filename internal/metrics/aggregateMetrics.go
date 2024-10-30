package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// tcp
	tcpCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "packetsleuth_packet_tcp_count",
		Help: "Number of TCP packets",
	})
	// syn-flood
	synAckCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_tcp_syn_ack_packets",
		Help: "Total number of TCP SYN and SYN-ACK packets",
	}, []string{"type"})
	// udp
	udpCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "packetsleuth_packet_udp_count",
		Help: "Number of UDP packets",
	})
	// all packet
	packetCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "packetsleuth_packet_count",
		Help: "Number of all packets",
	})
	// количество пакетов в привязке к порту отправителя
	portSrc = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_port_src",
		Help: "The number of packets from a specific port.",
	}, []string{"source_ports"})
	// количество пакетов в привязке к порту получателя
	portDst = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_port_dst",
		Help: "The number of packets to a specific port.",
	}, []string{"dest_port"})
	// количество пакетов по IP-адресу отправителя
	ipSrc = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_ip_src",
		Help: "The number of packets from a specific IP address.",
	}, []string{"src_ip"})
	// количество пакетов по IP-адресу получателя
	ipDst = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_ip_dst",
		Help: "The number of packets to a specific IP address.",
	}, []string{"dest_ip"})
	// общий объем трафика B/s (Bps)
	trafficTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "packetsleuth_traffic_total_bps",
		Help: "Total traffic in bytes per second.",
	})
	// объем трафика по исходящим портам
	trafficPortSrc = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_traffic_port_src_bps",
		Help: "Traffic in bytes per second from a specific source port.",
	}, []string{"source_port"})
	// объем трафика по входящим портам
	trafficPortDst = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "packetsleuth_traffic_port_dst_bps",
		Help: "Traffic in bytes per second to a specific destination port.",
	}, []string{"dest_port"})
)

type Monitor struct {
	packetChan   <-chan Packet
	influxClient influxdb2.Client
	writeAPI     api.WriteAPIBlocking
}

func NewMonitor(c <-chan Packet, influxURL, influxUser, influxPass, influxOrg, influxBucket string) *Monitor {
	influxclient := influxdb2.NewClient(influxURL, fmt.Sprintf("%s:%s", influxUser, influxPass))
	return &Monitor{
		packetChan:   c,
		influxClient: influxclient,
		writeAPI:     influxclient.WriteAPIBlocking(influxOrg, influxBucket),
	}
}

func (m *Monitor) Start() {
	chP := make(chan Packet, 100)
	go m.Prometheus(chP)
	for p := range m.packetChan {
		chP <- p
	}
	defer m.influxClient.Close()
}

func (m *Monitor) Prometheus(cp <-chan Packet) {
	for p := range cp {
		packetCount.Inc()
		trafficTotal.Add(float64(p.PayloadSize))
		if p.TransportType == TCP {
			tcpCount.Inc()
			if p.IsSYN && p.IsSYNACK {
				synAckCounterVec.WithLabelValues("syn_ack").Inc()
			} else if p.IsSYN {
				synAckCounterVec.WithLabelValues("syn").Inc()
			}
			portSrc.WithLabelValues(string(p.SrcPort)).Inc()
			portDst.WithLabelValues(string(p.DstPort)).Inc()
			trafficPortSrc.WithLabelValues(string(p.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(string(p.DstPort)).Add(float64(p.PayloadSize))
		}
		if p.TransportType == UDP {
			udpCount.Inc()
			portSrc.WithLabelValues(string(p.SrcPort)).Inc()
			portDst.WithLabelValues(string(p.DstPort)).Inc()
			trafficPortSrc.WithLabelValues(string(p.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(string(p.DstPort)).Add(float64(p.PayloadSize))
		}
		if p.SCTPInfo != nil {
			portSrc.WithLabelValues(string(p.SCTPInfo.SrcPort)).Inc()
			portDst.WithLabelValues(string(p.SCTPInfo.DstPort)).Inc()

			trafficPortSrc.WithLabelValues(string(p.SCTPInfo.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(string(p.SCTPInfo.DstPort)).Add(float64(p.PayloadSize))
		}

		if p.NetworkType == IPv4 || p.NetworkType == IPv6 {
			ipSrc.WithLabelValues(p.SrcIP.String()).Inc()
			ipDst.WithLabelValues(p.DstIP.String()).Inc()
		}
	}
}

func (m *Monitor) monitoringPacket(cp <-chan Packet) {
	for p := range cp {
		tags := map[string]string{
			"src_port": fmt.Sprint(p.SrcPort),
			"dst_port": fmt.Sprint(p.DstPort),
			"src_ip":   p.SrcIP.String(),
			"dst_ip":   p.DstIP.String(),
		}
		fields := map[string]interface{}{
			"packet_count":  1,
			"payload_size":  float64(p.PayloadSize),
			"tcp_count":     0,
			"udp_count":     0,
			"traffic_total": float64(p.PayloadSize),
		}

		if p.TransportType == TCP {
			fields["tcp_count"] = 1
		}
		if p.TransportType == UDP {
			fields["udp_count"] = 1
		}

		point := influxdb2.NewPoint(
			"packetsleuth_metrics",
			tags,
			fields,
			time.Now(), // Временная метка
		)
		err := m.writeAPI.WritePoint(context.Background(), point)
		if err != nil {
			log.Printf("Error writing to InfluxDB: %v", err)
		}
	}
}
