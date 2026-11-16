package notify

import (
	"strings"
	"testing"
	"time"

	"monitoring_server/internal/battery"
)

func TestBuildMessageContainsBatteryDetails(t *testing.T) {
	now := time.Date(2026, 11, 16, 14, 32, 0, 0, time.UTC)
	info := battery.BatteryInfo{Percentage: 14.5, Status: "Discharging"}
	msg := string(buildMessage("me@gmail.com", "me@gmail.com", "srvbox", info, now))

	for _, want := range []string{"To: me@gmail.com", "srvbox", "14.5", "Discharging"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "secretpass") {
		t.Fatal("password must not be in the email")
	}
}
