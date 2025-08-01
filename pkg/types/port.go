package types

type Transport string

const (
	TransportTCP Transport = "tcp"
	TransportUDP Transport = "udp"
)

type Port struct {
	Transport Transport
	Port      string
}

func NewPort(transport, port string) *Port {
	return &Port{
		Transport: Transport(transport),
		Port:      port,
	}
}
