package battery

type BatteryInfo struct {
	Percentage   float64 `json:"percentage"`
	Status       string  `json:"status"`        // "Charging", "Discharging", "Full", "Unknown", "Idle", "Empty"
	EnergyNow    int64   `json:"energy_now"`    // mWh
	EnergyFull   int64   `json:"energy_full"`   // mWh
	EnergyDesign int64   `json:"energy_design"` // mWh
	Voltage      int64   `json:"voltage"`       // mV
	Health       float64 `json:"health"`        // percentage of design capacity
}
