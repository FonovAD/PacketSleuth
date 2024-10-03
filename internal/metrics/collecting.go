package metrics

import (
	"log"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	PacketMaxSize      = 1600
	PacketChanSize     = 100
	Ethernet           = "Ethernet"
	IPv4               = "IPv4"
	IPv6               = "IPv6"
	UnknownNetworkType = "Unknown"
	TCP                = "TCP"
	UDP                = "UDP"
	ARP                = "ARP"
	DNS                = "DNS"
	SCTP               = "SCTP"
	HTTP               = "HTTP"
)

type MetricMonitor struct {
	devices []pcap.Interface
}

type Packet struct {
	TimeStamp     time.Time
	LinkType      string
	SrcMAC        net.HardwareAddr
	DstMAC        net.HardwareAddr
	NetworkType   string
	SrcIP         net.IP
	DstIP         net.IP
	TransportType string
	SrcPort       uint16
	DstPort       uint16
	PayloadSize   int
	Application   string
	IsMalformed   bool
	ARPInfo       *ARPInfo
	SCTPInfo      *SCTPInfo
	IsSYN         bool
	IsSYNACK      bool
}

type ARPInfo struct {
	SenderIP  net.IP
	SenderMAC net.HardwareAddr
	TargetIP  net.IP
	TargetMAC net.HardwareAddr
}

type SCTPInfo struct {
	SrcPort uint16
	DstPort uint16
}

func NewMetricMonitor() *MetricMonitor {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	return &MetricMonitor{
		devices: devices,
	}
}

func (m *MetricMonitor) Listen() <-chan Packet {
	pchan := make(chan Packet, PacketChanSize)
	for _, device := range m.devices {
		go capturePackets(device.Name, pchan)
	}
	lock := make(chan bool)
	lock <- true
	return pchan
}

func capturePackets(deviceName string, pchan chan<- Packet) {
	handle, err := pcap.OpenLive(deviceName, PacketMaxSize, true, pcap.BlockForever)
	if err != nil {
		log.Printf("Error opening the device %s: %v", deviceName, err)
		return
	}
	defer handle.Close()
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		packetInfo := processPacket(packet)
		if packetInfo != nil {
			pchan <- *packetInfo
		}
	}
}

func getDNSLayer(packet gopacket.Packet) *layers.DNS {
	var dns layers.DNS
	err := dns.DecodeFromBytes(packet.Layer(layers.LayerTypeDNS).LayerContents(), gopacket.NilDecodeFeedback)
	if err == nil {
		return &dns
	}
	return nil
}

func processPacket(packet gopacket.Packet) *Packet {
	packetInfo := &Packet{}

	linkLayer := packet.LinkLayer()
	if linkLayer != nil {
		switch layer := linkLayer.(type) {
		case *layers.Ethernet:
			packetInfo.LinkType = Ethernet
			packetInfo.SrcMAC = layer.SrcMAC
			packetInfo.DstMAC = layer.DstMAC
		default:
			packetInfo.LinkType = UnknownNetworkType
		}
	}

	networkLayer := packet.NetworkLayer()
	if networkLayer != nil {
		switch layer := networkLayer.(type) {
		case *layers.IPv4:
			packetInfo.NetworkType = IPv4
			packetInfo.SrcIP = layer.SrcIP
			packetInfo.DstIP = layer.DstIP
		case *layers.IPv6:
			packetInfo.NetworkType = IPv6
			packetInfo.SrcIP = layer.SrcIP
			packetInfo.DstIP = layer.DstIP
		default:
			packetInfo.NetworkType = UnknownNetworkType
		}
	}

	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		arp, _ := arpLayer.(*layers.ARP)
		packetInfo.NetworkType = ARP
		arpInfo := &ARPInfo{
			SenderIP:  net.IP(arp.SourceProtAddress),
			SenderMAC: net.HardwareAddr(arp.SourceHwAddress),
			TargetIP:  net.IP(arp.DstProtAddress),
			TargetMAC: net.HardwareAddr(arp.DstHwAddress),
		}
		packetInfo.ARPInfo = arpInfo
	}

	transportLayer := packet.TransportLayer()
	if transportLayer != nil {
		switch layer := transportLayer.(type) {
		case *layers.TCP:
			packetInfo.TransportType = TCP
			packetInfo.SrcPort = uint16(layer.SrcPort)
			packetInfo.DstPort = uint16(layer.DstPort)
			packetInfo.PayloadSize = len(layer.Payload)

			packetInfo.IsSYN = layer.SYN
			packetInfo.IsSYNACK = layer.SYN && layer.ACK

			if packetInfo.SrcPort == 80 || packetInfo.DstPort == 80 || packetInfo.SrcPort == 443 || packetInfo.DstPort == 443 {
				packetInfo.Application = HTTP
			}
		case *layers.UDP:
			packetInfo.TransportType = UDP
			packetInfo.SrcPort = uint16(layer.SrcPort)
			packetInfo.DstPort = uint16(layer.DstPort)
			packetInfo.PayloadSize = len(layer.Payload)

			if packetInfo.SrcPort == 53 || packetInfo.DstPort == 53 {
				if dnsLayer := getDNSLayer(packet); dnsLayer != nil {
					packetInfo.Application = DNS
				}
			}
		case *layers.SCTP:
			packetInfo.TransportType = SCTP
			sctpInfo := &SCTPInfo{
				SrcPort: uint16(layer.SrcPort),
				DstPort: uint16(layer.DstPort),
			}
			packetInfo.SCTPInfo = sctpInfo
		default:
			packetInfo.TransportType = UnknownNetworkType
		}
	} else {
		packetInfo.IsMalformed = true
	}
	if packet.ErrorLayer() != nil {
		packetInfo.IsMalformed = true
	}
	packetInfo.TimeStamp = time.Now()
	return packetInfo
}
