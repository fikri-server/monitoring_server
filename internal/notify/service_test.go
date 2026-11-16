package notify

import (
	"errors"
	"strings"
	"testing"
	"time"

	"monitoring_server/internal/battery"
)

func testConfig() Config {
	return Config{User: "me@gmail.com", Password: "secretpass", Recipient: "me@gmail.com", Threshold: 20}
}

func countingSender(calls *int, err error) Sender {
	return func(user, password, to string, msg []byte) error {
		*calls++
		return err
	}
}

func TestCheckSendsOncePerEpisode(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, nil))
	now := time.Now()
	low := battery.BatteryInfo{Percentage: 15, Status: "Discharging"}

	s.Check(low, now)
	s.Check(low, now.Add(10*time.Second))
	s.Check(low, now.Add(20*time.Second))
	if calls != 1 {
		t.Fatalf("expected 1 email per episode, got %d", calls)
	}
	if s.Status().LastSent == "" {
		t.Fatal("last_sent should be set")
	}
	if s.Status().Armed {
		t.Fatal("should not be armed after sending")
	}
}

func TestCheckResetsOnChargingThenSendsAgain(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, nil))
	now := time.Now()
	low := battery.BatteryInfo{Percentage: 15, Status: "Discharging"}

	s.Check(low, now)
	s.Check(battery.BatteryInfo{Percentage: 15, Status: "Charging"}, now.Add(time.Second))
	s.Check(low, now.Add(2*time.Second))
	if calls != 2 {
		t.Fatalf("expected a new email after a new episode, got %d", calls)
	}
}

func TestCheckResetsWhenBackAboveThreshold(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, nil))
	now := time.Now()

	s.Check(battery.BatteryInfo{Percentage: 15, Status: "Discharging"}, now)
	s.Check(battery.BatteryInfo{Percentage: 25, Status: "Discharging"}, now.Add(time.Second))
	s.Check(battery.BatteryInfo{Percentage: 19, Status: "Discharging"}, now.Add(2*time.Second))
	if calls != 2 {
		t.Fatalf("expected reset above threshold, got %d calls", calls)
	}
}

func TestCheckIgnoresMissingBattery(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, nil))
	s.Check(battery.BatteryInfo{}, time.Now())
	if calls != 0 {
		t.Fatalf("zero-value battery must not send, got %d", calls)
	}
}

func TestCheckNotConfiguredDoesNothing(t *testing.T) {
	calls := 0
	s := NewService(Config{Threshold: 20}, countingSender(&calls, nil))
	s.Check(battery.BatteryInfo{Percentage: 5, Status: "Discharging"}, time.Now())
	if calls != 0 {
		t.Fatalf("not configured must not send, got %d", calls)
	}
}

func TestCheckRetriesAfterCooldownOnFailure(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, errors.New("smtp down")))
	now := time.Now()
	low := battery.BatteryInfo{Percentage: 15, Status: "Discharging"}

	s.Check(low, now)
	s.Check(low, now.Add(2*time.Second))
	if calls != 1 {
		t.Fatalf("must wait for cooldown, got %d calls", calls)
	}
	s.Check(low, now.Add(61*time.Second))
	if calls != 2 {
		t.Fatalf("expected retry after 1 minute, got %d calls", calls)
	}
	if s.Status().LastError != "smtp down" {
		t.Fatalf("expected last error, got %q", s.Status().LastError)
	}
}

func TestStatusInitial(t *testing.T) {
	s := NewService(testConfig(), countingSender(new(int), nil))
	st := s.Status()
	if !st.Configured || !st.Armed || st.Threshold != 20 || st.LastSent != "" || st.LastError != "" {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestSanitizeRemovesSecrets(t *testing.T) {
	got := sanitize("550 5.1.1 me@gmail.com rejected, pass secretpass", "me@gmail.com", "secretpass")
	if strings.Contains(got, "me@gmail.com") || strings.Contains(got, "secretpass") {
		t.Fatalf("secrets leaked: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redaction marker: %q", got)
	}
}
