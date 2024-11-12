package metrics

import (
	"fmt"
	"time"

	localStore "github.com/FonovAD/PacketSleuth/internal/store/LocalStore"

	"github.com/FonovAD/PacketSleuth/internal/store"
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

type Config struct {
	UsePrometheus bool
}

type Monitor struct {
	packetChan <-chan Packet
	localStore store.Store
}

func NewMonitor(c <-chan Packet, dbName string) *Monitor {
	if dbName != "" {
		storeL := localStore.InitSqlStore(dbName)
		p := Packet{}
		pMap := make(map[string]interface{})
		pMap["LinkType"] = p.LinkType
		pMap["SrcMAC"] = p.SrcMAC
		pMap["DstMAC"] = p.DstMAC
		pMap["NetworkType"] = p.NetworkType
		pMap["SrcIP"] = p.SrcIP
		pMap["DstIP"] = p.DstIP
		pMap["TransportType"] = p.TransportType
		pMap["SrcPort"] = p.SrcPort
		pMap["DstPort"] = p.DstPort
		pMap["PayloadSize"] = p.PayloadSize
		pMap["Application"] = p.Application
		pMap["IsMalformed"] = p.IsMalformed
		pMap["ARPInfo"] = p.ARPInfo
		pMap["SCTPInfo"] = p.SCTPInfo
		pMap["IsSYN"] = p.IsSYN
		pMap["IsSYNACK"] = p.IsSYNACK
		storeL.CreateTable(*store.NewPoint(time.Now(), pMap))
		return &Monitor{
			packetChan: c,
			localStore: storeL,
		}
	} else {
		return &Monitor{
			packetChan: c,
		}
	}
}

func (m *Monitor) Start(cfg Config) {
	var chP chan Packet
	var chL chan Packet
	if cfg.UsePrometheus {
		chP := make(chan Packet, 100)
		go m.collectPrometheus(chP)
		defer close(chP)
	}
	if m.localStore != nil {
		chL := make(chan Packet, 100)
		go m.Local(chL)
		defer close(chL)
		defer m.localStore.Close()
	}
	for p := range m.packetChan {
		if cfg.UsePrometheus {
			chP <- p
		}
		if m.localStore != nil {
			chL <- p
		}
	}
}

func (m *Monitor) collectPrometheus(cp <-chan Packet) {
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
			portSrc.WithLabelValues(fmt.Sprint(p.SrcPort)).Inc()
			portDst.WithLabelValues(fmt.Sprint(p.DstPort)).Inc()
			trafficPortSrc.WithLabelValues(fmt.Sprint(p.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(fmt.Sprint(p.DstPort)).Add(float64(p.PayloadSize))
		}
		if p.TransportType == UDP {
			udpCount.Inc()
			portSrc.WithLabelValues(fmt.Sprint(p.SrcPort)).Inc()
			portDst.WithLabelValues(fmt.Sprint(p.DstPort)).Inc()
			trafficPortSrc.WithLabelValues(fmt.Sprint(p.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(fmt.Sprint(p.DstPort)).Add(float64(p.PayloadSize))
		}
		if p.SCTPInfo != nil {
			portSrc.WithLabelValues(fmt.Sprint(p.SCTPInfo.SrcPort)).Inc()
			portDst.WithLabelValues(fmt.Sprint(p.SCTPInfo.DstPort)).Inc()

			trafficPortSrc.WithLabelValues(fmt.Sprint(p.SCTPInfo.SrcPort)).Add(float64(p.PayloadSize))
			trafficPortDst.WithLabelValues(fmt.Sprint(p.SCTPInfo.DstPort)).Add(float64(p.PayloadSize))
		}

		if p.NetworkType == IPv4 || p.NetworkType == IPv6 {
			ipSrc.WithLabelValues(p.SrcIP.String()).Inc()
			ipDst.WithLabelValues(p.DstIP.String()).Inc()
		}
	}
}

// нужно переделать метод хранения метрик. Нужно хранить сразу подсчитанное значение за секунду.
// Это ускорит программу(меньше )
func (m *Monitor) Local(cp <-chan Packet) {
	pMap := make(map[string]interface{})
	for p := range cp {
		pMap["LinkType"] = p.LinkType
		pMap["SrcMAC"] = p.SrcMAC
		pMap["DstMAC"] = p.DstMAC
		pMap["NetworkType"] = p.NetworkType
		pMap["SrcIP"] = p.SrcIP
		pMap["DstIP"] = p.DstIP
		pMap["TransportType"] = p.TransportType
		pMap["SrcPort"] = p.SrcPort
		pMap["DstPort"] = p.DstPort
		pMap["PayloadSize"] = p.PayloadSize
		pMap["Application"] = p.Application
		pMap["IsMalformed"] = p.IsMalformed
		pMap["ARPInfo"] = p.ARPInfo
		pMap["SCTPInfo"] = p.SCTPInfo
		pMap["IsSYN"] = p.IsSYN
		pMap["IsSYNACK"] = p.IsSYNACK
		m.localStore.WritePoint(*store.NewPoint(p.TimeStamp, pMap))
	}
}
