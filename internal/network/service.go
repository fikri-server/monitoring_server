package network

import (
	"github.com/shirou/gopsutil/v3/net"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetNetInfo() (NetInfo, error) {
	netInfo, _ := net.IOCounters(false)
	connections, _ := net.Connections("all")

	result := NetInfo{
		Connections: len(connections),
	}

	if len(netInfo) > 0 {
		result.BytesSent = netInfo[0].BytesSent
		result.BytesRecv = netInfo[0].BytesRecv
	}

	return result, nil
}
