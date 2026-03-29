package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"monitoring_server/internal/battery"
	"monitoring_server/internal/cpu"
	"monitoring_server/internal/disk"
	"monitoring_server/internal/memory"
	"monitoring_server/internal/network"
	"monitoring_server/internal/sysinfo"
)

type SystemInfo struct {
	Hostname    string              `json:"hostname"`
	OS          string              `json:"os"`
	Platform    string              `json:"platform"`
	PlatformVer string              `json:"platform_version"`
	Uptime      uint64              `json:"uptime"`
	CPU         cpu.CPUInfo         `json:"cpu"`
	Memory      memory.MemInfo      `json:"memory"`
	Disk        disk.DiskInfo       `json:"disk"`
	Network     network.NetInfo     `json:"network"`
	Temperature float64             `json:"temperature"`
	LoadAvg     sysinfo.LoadInfo    `json:"load_average"`
	Processes   int                 `json:"processes"`
	Battery     battery.BatteryInfo `json:"battery"` // <-- field battery
	LastUpdate  time.Time           `json:"last_update"`
}

var systemInfo SystemInfo

func main() {
	// Inisialisasi services
	cpuService := cpu.NewService()
	memService := memory.NewService()
	diskService := disk.NewService("/")
	netService := network.NewService()
	sysService := sysinfo.NewService()
	batteryService := battery.NewService()

	// Inisialisasi handlers
	cpuHandler := cpu.NewHandler(cpuService)
	memHandler := memory.NewHandler(memService)
	diskHandler := disk.NewHandler(diskService)
	netHandler := network.NewHandler(netService)
	sysHandler := sysinfo.NewHandler(sysService)
	batteryHandler := battery.NewHandler(batteryService)

	// Background updater
	go updateAllData(cpuService, memService, diskService, netService, sysService, batteryService)

	// Routes
	http.HandleFunc("/", homePage)
	http.HandleFunc("/api/monitor", apiMonitor)
	http.HandleFunc("/api/cpu", cpuHandler.GetCPUInfoHandler)
	http.HandleFunc("/api/memory", memHandler.GetMemInfoHandler)
	http.HandleFunc("/api/disk", diskHandler.GetDiskInfoHandler)
	http.HandleFunc("/api/network", netHandler.GetNetInfoHandler)
	http.HandleFunc("/api/system", sysHandler.GetSystemInfoHandler)
	http.HandleFunc("/api/battery", batteryHandler.GetBatteryInfoHandler)
	http.HandleFunc("/api/battery/percentage", batteryHandler.GetBatteryPercentageHandler)
	http.HandleFunc("/api/battery/status", batteryHandler.GetBatteryStatusHandler)
	http.HandleFunc("/api/battery/health", batteryHandler.GetBatteryHealthHandler)
	http.HandleFunc("/health", healthCheck)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := ":8081"
	log.Printf("🚀 Monitoring Server berjalan di http://localhost%s", port)
	log.Println("📊 Monitor resource server secara real-time")
	log.Printf("📍 Endpoints: /, /api/monitor, /api/cpu, /api/memory, /api/disk, /api/network, /api/system, /api/battery*, /health")
	log.Fatal(http.ListenAndServe(port, nil))
}

func updateAllData(
	cpuService *cpu.Service,
	memService *memory.Service,
	diskService *disk.Service,
	netService *network.Service,
	sysService *sysinfo.Service,
	batteryService *battery.Service,
) {
	for {
		cpuInfo, _ := cpuService.GetCPUInfo()
		memInfo, _ := memService.GetMemInfo()
		diskInfo, _ := diskService.GetDiskInfo()
		netInfo, _ := netService.GetNetInfo()
		sysInfo, _ := sysService.GetSystemInfo()
		batteryInfo, _ := batteryService.GetBatteryInfo() // bisa error, tapi akan tetap zero value

		systemInfo = SystemInfo{
			Hostname:    sysInfo.Hostname,
			OS:          sysInfo.OS,
			Platform:    sysInfo.Platform,
			PlatformVer: sysInfo.PlatformVer,
			Uptime:      sysInfo.Uptime,
			CPU:         cpuInfo,
			Memory:      memInfo,
			Disk:        diskInfo,
			Network:     netInfo,
			Temperature: sysInfo.Temperature,
			LoadAvg:     sysInfo.LoadAvg,
			Processes:   sysInfo.Processes,
			Battery:     batteryInfo, // akan tetap diisi meskipun kosong
			LastUpdate:  time.Now(),
		}

		time.Sleep(2 * time.Second)
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func apiMonitor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(systemInfo)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
