package battery

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const sysfs = "/sys/class/power_supply"

type Service struct {
	lastCheck time.Time
}

func NewService() *Service {
	return &Service{}
}

// readFloat membaca file dan mengembalikan nilai float64 (dalam mWh atau mV)
// Konversi dari micro ke milli jika perlu (dibagi 1000)
func readFloat(path, filename string) (float64, error) {
	data, err := ioutil.ReadFile(filepath.Join(path, filename))
	if err != nil {
		return 0, err
	}
	// Hapus newline dan spasi
	str := strings.TrimSpace(string(data))
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	// Biasanya nilai dalam micro, konversi ke milli
	return num / 1000, nil
}

// readAmp membaca arus (current) dalam mW = ampere * voltage
func readAmp(path, filename string, volts float64) (float64, error) {
	val, err := readFloat(path, filename)
	if err != nil {
		return 0, err
	}
	return val * volts, nil
}

// isBattery mengecek apakah direktori adalah baterai
func isBattery(path string) bool {
	data, err := ioutil.ReadFile(filepath.Join(path, "type"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "Battery"
}

// getBatteryPaths mengembalikan daftar path baterai
func getBatteryPaths() ([]string, error) {
	files, err := ioutil.ReadDir(sysfs)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, file := range files {
		path := filepath.Join(sysfs, file.Name())
		if isBattery(path) {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// GetBatteryInfo mengambil info baterai pertama yang ditemukan
func (s *Service) GetBatteryInfo() (BatteryInfo, error) {
	paths, err := getBatteryPaths()
	if err != nil {
		return BatteryInfo{}, fmt.Errorf("failed to read battery paths: %v", err)
	}
	if len(paths) == 0 {
		return BatteryInfo{}, fmt.Errorf("no battery found")
	}
	path := paths[0]

	var (
		current    float64 // mWh
		full       float64 // mWh
		design     float64 // mWh
		voltage    float64 // V
		chargeRate float64 // mW
		statusStr  string
	)

	// Baca energy_now atau charge_now
	energyNow, err := readFloat(path, "energy_now")
	if err != nil && os.IsNotExist(err) {
		// Fallback ke charge_now
		voltage, _ = readFloat(path, "voltage_now")
		voltage /= 1000 // konversi ke Volt
		chargeNow, err := readFloat(path, "charge_now")
		if err == nil {
			current = chargeNow * voltage
		}
		fullCharge, err := readFloat(path, "charge_full")
		if err == nil {
			full = fullCharge * voltage
		}
		designCharge, err := readFloat(path, "charge_full_design")
		if err == nil {
			design = designCharge * voltage
		}
		chargeRate, err = readFloat(path, "current_now")
		if err == nil {
			chargeRate = chargeRate * voltage
		}
	} else {
		current = energyNow
		full, _ = readFloat(path, "energy_full")
		design, _ = readFloat(path, "energy_full_design")
		chargeRate, _ = readFloat(path, "power_now")
		voltage, _ = readFloat(path, "voltage_now")
		voltage /= 1000 // konversi ke Volt
	}

	// Baca status
	statusData, err := ioutil.ReadFile(filepath.Join(path, "status"))
	if err == nil {
		statusStr = strings.TrimSpace(string(statusData))
		// Normalisasi status
		switch statusStr {
		case "Not charging":
			statusStr = "Idle"
		}
	} else {
		statusStr = "Unknown"
	}

	// Hitung persentase
	var percentage float64
	if full > 0 {
		percentage = (current / full) * 100
	}

	// Hitung health
	var health float64
	if design > 0 {
		health = (full / design) * 100
	}

	s.lastCheck = time.Now()

	return BatteryInfo{
		Percentage:   percentage,
		Status:       statusStr,
		EnergyNow:    int64(current),
		EnergyFull:   int64(full),
		EnergyDesign: int64(design),
		Voltage:      int64(voltage * 1000), // kembali ke mV
		Health:       health,
	}, nil
}

// GetBatteryPercentage mengambil persentase baterai
func (s *Service) GetBatteryPercentage() (float64, error) {
	info, err := s.GetBatteryInfo()
	if err != nil {
		return 0, err
	}
	return info.Percentage, nil
}

// GetBatteryStatus mengambil status baterai
func (s *Service) GetBatteryStatus() (string, error) {
	info, err := s.GetBatteryInfo()
	if err != nil {
		return "", err
	}
	return info.Status, nil
}

// IsCharging mengecek apakah baterai sedang dicharge
func (s *Service) IsCharging() (bool, error) {
	info, err := s.GetBatteryInfo()
	if err != nil {
		return false, err
	}
	return info.Status == "Charging", nil
}

// GetBatteryHealth mengambil kesehatan baterai
func (s *Service) GetBatteryHealth() (float64, error) {
	info, err := s.GetBatteryInfo()
	if err != nil {
		return 0, err
	}
	return info.Health, nil
}

// GetLastCheck mengembalikan waktu terakhir cek
func (s *Service) GetLastCheck() time.Time {
	return s.lastCheck
}
