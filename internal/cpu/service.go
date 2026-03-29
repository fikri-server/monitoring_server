package cpu

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetCPUInfo() (CPUInfo, error) {
	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuInfo, _ := cpu.Info()

	var percent float64
	if len(cpuPercent) > 0 {
		percent = cpuPercent[0]
	}

	var modelName string
	var cores int
	if len(cpuInfo) > 0 {
		modelName = cpuInfo[0].ModelName
		cores = len(cpuInfo)
	}

	return CPUInfo{
		Percent: percent,
		Cores:   cores,
		Model:   modelName,
	}, nil
}
