package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetStatusHidesCredentials(t *testing.T) {
	h := NewHandler(NewService(testConfig(), countingSender(new(int), nil)))
	rec := httptest.NewRecorder()
	h.GetStatus(rec, httptest.NewRequest(http.MethodGet, "/api/notify", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, secret := range []string{"me@gmail.com", "secretpass"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaks %q: %s", secret, body)
		}
	}
	var st Status
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if st.Threshold != 20 || !st.Configured || !st.Armed {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestGetStatusRejectsPost(t *testing.T) {
	h := NewHandler(NewService(testConfig(), countingSender(new(int), nil)))
	rec := httptest.NewRecorder()
	h.GetStatus(rec, httptest.NewRequest(http.MethodPost, "/api/notify", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
