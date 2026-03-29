package disk

import (
	"github.com/shirou/gopsutil/v3/disk"
)

type Service struct {
	path string
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func (s *Service) GetDiskInfo() (DiskInfo, error) {
	diskInfo, _ := disk.Usage(s.path)

	return DiskInfo{
		Total:       diskInfo.Total,
		Used:        diskInfo.Used,
		Free:        diskInfo.Free,
		UsedPercent: diskInfo.UsedPercent,
	}, nil
}
