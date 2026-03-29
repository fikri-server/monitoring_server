package memory

import (
	"github.com/shirou/gopsutil/v3/mem"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetMemInfo() (MemInfo, error) {
	memInfo, _ := mem.VirtualMemory()

	return MemInfo{
		Total:       memInfo.Total,
		Used:        memInfo.Used,
		Free:        memInfo.Free,
		UsedPercent: memInfo.UsedPercent,
	}, nil
}
