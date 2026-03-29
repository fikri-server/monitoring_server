package sysinfo

import "time"

type LoadInfo struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type SystemInfo struct {
	Hostname    string    `json:"hostname"`
	OS          string    `json:"os"`
	Platform    string    `json:"platform"`
	PlatformVer string    `json:"platform_version"`
	Uptime      uint64    `json:"uptime"`
	UptimeHuman string    `json:"uptime_human"`
	Temperature float64   `json:"temperature"`
	Processes   int       `json:"processes"`
	LastUpdate  time.Time `json:"last_update"`
	LoadAvg     LoadInfo  `json:"load_average"`
}
