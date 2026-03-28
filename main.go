package main

import (
    "encoding/json"
    "html/template"
    "fmt"
    "log"
    "net/http"
    "time"
    "os"

    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/load"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/net"
    "github.com/shirou/gopsutil/v3/process"
)

type SystemInfo struct {
    Hostname     string    `json:"hostname"`
    OS           string    `json:"os"`
    Platform     string    `json:"platform"`
    PlatformVer  string    `json:"platform_version"`
    Uptime       uint64    `json:"uptime"`
    CPU          CPUInfo   `json:"cpu"`
    Memory       MemInfo   `json:"memory"`
    Disk         DiskInfo  `json:"disk"`
    Network      NetInfo   `json:"network"`
    Temperature  float64   `json:"temperature"`
    LoadAvg      LoadInfo  `json:"load_average"`
    Processes    int       `json:"processes"`
    LastUpdate   time.Time `json:"last_update"`
}

type CPUInfo struct {
    Percent  float64   `json:"percent"`
    Cores    int       `json:"cores"`
    Model    string    `json:"model"`
}

type MemInfo struct {
    Total       uint64  `json:"total"`
    Used        uint64  `json:"used"`
    Free        uint64  `json:"free"`
    UsedPercent float64 `json:"used_percent"`
}

type DiskInfo struct {
    Total       uint64  `json:"total"`
    Used        uint64  `json:"used"`
    Free        uint64  `json:"free"`
    UsedPercent float64 `json:"used_percent"`
}

type NetInfo struct {
    BytesSent   uint64 `json:"bytes_sent"`
    BytesRecv   uint64 `json:"bytes_recv"`
    Connections int    `json:"connections"`
}

type LoadInfo struct {
    Load1  float64 `json:"load1"`
    Load5  float64 `json:"load5"`
    Load15 float64 `json:"load15"`
}

var systemInfo SystemInfo

func main() {
    // Update system info periodically
    go updateSystemInfo()

    // Routes
    http.HandleFunc("/", homePage)
    http.HandleFunc("/api/monitor", apiMonitor)
    http.HandleFunc("/api/cpu", apiCPU)
    http.HandleFunc("/api/memory", apiMemory)
    http.HandleFunc("/api/disk", apiDisk)
    http.HandleFunc("/api/system", apiSystem)
    http.HandleFunc("/health", healthCheck)

    // Serve static files
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    port := ":8081"
    log.Printf("🚀 Monitoring Server berjalan di http://localhost%s", port)
    log.Println("📊 Monitor resource server secara real-time")
    log.Printf("📍 Endpoints: /, /api/monitor, /api/cpu, /api/memory, /api/disk, /api/system, /health")
    log.Fatal(http.ListenAndServe(port, nil))
}

func updateSystemInfo() {
    for {
        // Get host info
        hostInfo, _ := host.Info()
        
        // Get CPU info
        cpuPercent, _ := cpu.Percent(time.Second, false)
        cpuInfo, _ := cpu.Info()
        
        // Get memory info
        memInfo, _ := mem.VirtualMemory()
        
        // Get disk info
        diskInfo, _ := disk.Usage("/")
        
        // Get network info
        netInfo, _ := net.IOCounters(false)
        
        // Get temperature using host.SensorsTemperatures
        var temperature float64
        tempSensors, err := host.SensorsTemperatures()
        if err == nil && len(tempSensors) > 0 {
            // Cari sensor dengan suhu yang valid (biasanya yang pertama)
            for _, s := range tempSensors {
                if s.Temperature > 0 {
                    temperature = s.Temperature
                    break
                }
            }
        }
        
        // Alternative: kalau host.SensorsTemperatures tidak berhasil,
        // kita bisa baca dari file system thermal zone
        if temperature == 0 {
            temperature = readThermalZone()
        }
        
        // Get load average
        loadAvg, err := load.Avg()
        var loadInfo LoadInfo
        if err == nil {
            loadInfo = LoadInfo{
                Load1:  loadAvg.Load1,
                Load5:  loadAvg.Load5,
                Load15: loadAvg.Load15,
            }
        }
        
        // Get process count
        processes, _ := process.Processes()
        processCount := len(processes)
        
        systemInfo = SystemInfo{
            Hostname:    hostInfo.Hostname,
            OS:          hostInfo.OS,
            Platform:    hostInfo.Platform,
            PlatformVer: hostInfo.PlatformVersion,
            Uptime:      hostInfo.Uptime,
            LastUpdate:  time.Now(),
            CPU: CPUInfo{
                Percent: cpuPercent[0],
                Cores:   len(cpuInfo),
                Model:   cpuInfo[0].ModelName,
            },
            Memory: MemInfo{
                Total:       memInfo.Total,
                Used:        memInfo.Used,
                Free:        memInfo.Free,
                UsedPercent: memInfo.UsedPercent,
            },
            Disk: DiskInfo{
                Total:       diskInfo.Total,
                Used:        diskInfo.Used,
                Free:        diskInfo.Free,
                UsedPercent: diskInfo.UsedPercent,
            },
            Temperature: temperature,
            LoadAvg:     loadInfo,
            Processes:   processCount,
        }
        
        if len(netInfo) > 0 {
            systemInfo.Network = NetInfo{
                BytesSent: netInfo[0].BytesSent,
                BytesRecv: netInfo[0].BytesRecv,
            }
        }
        
        time.Sleep(2 * time.Second)
    }
}

// Fungsi alternatif untuk membaca temperature dari thermal zone
func readThermalZone() float64 {
    // Coba baca dari /sys/class/thermal/thermal_zone*/temp
    zones := []string{
        "/sys/class/thermal/thermal_zone0/temp",
        "/sys/class/thermal/thermal_zone1/temp",
        "/sys/class/thermal/thermal_zone2/temp",
    }
    
    for _, zone := range zones {
        data, err := os.ReadFile(zone)
        if err == nil {
            var temp int64
            fmt.Sscanf(string(data), "%d", &temp)
            if temp > 0 {
                // Temperature dalam mili derajat Celsius
                return float64(temp) / 1000.0
            }
        }
    }
    return 0
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

func apiCPU(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(systemInfo.CPU)
}

func apiMemory(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(systemInfo.Memory)
}

func apiDisk(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(systemInfo.Disk)
}

func apiSystem(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    
    systemData := map[string]interface{}{
        "hostname":      systemInfo.Hostname,
        "os":           systemInfo.OS,
        "platform":     systemInfo.Platform,
        "version":      systemInfo.PlatformVer,
        "uptime":       systemInfo.Uptime,
        "uptime_human": formatUptime(systemInfo.Uptime),
        "temperature":  systemInfo.Temperature,
        "processes":    systemInfo.Processes,
        "load_average": systemInfo.LoadAvg,
    }
    
    json.NewEncoder(w).Encode(systemData)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "time":   time.Now().Format(time.RFC3339),
    })
}

func formatUptime(seconds uint64) string {
    days := seconds / 86400
    hours := (seconds % 86400) / 3600
    minutes := (seconds % 3600) / 60
    
    if days > 0 {
        return formatDays(days, hours, minutes)
    } else if hours > 0 {
        return formatHours(hours, minutes)
    }
    return formatMinutes(minutes)
}

func formatDays(days, hours, minutes uint64) string {
    if days == 1 {
        if hours == 1 {
            return "1 day 1 hour"
        } else if hours > 0 {
            return "1 day " + formatHoursOnly(hours)
        }
        return "1 day"
    }
    
    if hours > 0 {
        if hours == 1 {
            return formatDaysOnly(days) + " 1 hour"
        }
        return formatDaysOnly(days) + " " + formatHoursOnly(hours)
    }
    return formatDaysOnly(days)
}

func formatHours(hours, minutes uint64) string {
    if hours == 1 {
        if minutes == 1 {
            return "1 hour 1 minute"
        } else if minutes > 0 {
            return "1 hour " + formatMinutesOnly(minutes)
        }
        return "1 hour"
    }
    
    if minutes > 0 {
        if minutes == 1 {
            return formatHoursOnly(hours) + " 1 minute"
        }
        return formatHoursOnly(hours) + " " + formatMinutesOnly(minutes)
    }
    return formatHoursOnly(hours)
}

func formatDaysOnly(days uint64) string {
    return itoa(days) + " days"
}

func formatHoursOnly(hours uint64) string {
    if hours == 1 {
        return "1 hour"
    }
    return itoa(hours) + " hours"
}

func formatMinutes(minutes uint64) string {
    if minutes == 1 {
        return "1 minute"
    }
    return itoa(minutes) + " minutes"
}

func formatMinutesOnly(minutes uint64) string {
    if minutes == 1 {
        return "1 minute"
    }
    return itoa(minutes) + " minutes"
}

func itoa(n uint64) string {
    if n == 0 {
        return "0"
    }
    // Simple conversion
    digits := []byte{}
    for n > 0 {
        digits = append([]byte{byte('0' + n%10)}, digits...)
        n /= 10
    }
    return string(digits)
}
