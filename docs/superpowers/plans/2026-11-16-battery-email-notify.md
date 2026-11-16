# Notifikasi Email Baterai Discharged — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Kirim email Gmail sekali per episode saat baterai `Discharging` di bawah threshold, dengan konfigurasi dari `.env` dan status read-only di dashboard tanpa membocorkan alamat email.

**Architecture:** Package baru `internal/notify` berisi pembacaan config dari environment, pembuatan pesan + pengiriman via `net/smtp`, state machine `Check()` yang dipanggil dari loop 2 detik di `main.go`, dan dua handler HTTP (`GET /api/notify`, `POST /api/notify/test`). Frontend menambahkan satu info-card yang hanya menampilkan status.

**Tech Stack:** Go 1.22, stdlib `net/smtp`, `github.com/joho/godotenv`, HTTP stdlib, JavaScript vanilla.

**Spec:** `docs/superpowers/specs/2026-11-16-battery-email-notify-design.md`

## Global Constraints

- Konfigurasi hanya dari environment: `SMTP_USER`, `SMTP_PASS`, `NOTIFY_TO` (default `SMTP_USER`), `NOTIFY_THRESHOLD` (default `20`).
- Tidak ada endpoint yang mengubah konfigurasi; tidak ada kredensial yang disimpan ke file.
- `SMTP_USER`, `SMTP_PASS`, dan `NOTIFY_TO` tidak boleh muncul di response API, HTML/DOM, atau log.
- Balasan API dan UI hanya memuat: `configured`, `threshold`, `last_sent`, `last_error`, `armed`.
- Email hanya dikirim sekali per episode discharge; retry maksimal tiap 1 menit.
- Bahasa salinan UI mengikuti dashboard yang ada (label Inggris pendek, nilai apa adanya).

## Review Focus

- Baterai tidak terbaca / tidak ada baterai (`BatteryInfo` zero value: `Percentage 0`, `Status ""`) — tidak boleh memicu email.
- Kondisi discharge bertahan — `Check` dipanggil ratusan kali, email harus tepat satu.
- Transisi charging lalu discharge lagi — episode baru harus mengirim email baru.
- Pengiriman gagal (WiFi mati) — tidak retry tiap 2 detik; retry setelah cooldown 1 menit.
- Error SMTP biasanya memuat alamat email atau respons server — harus disanitasi sebelum disimpan dan dikirim ke UI.
- `NOTIFY_THRESHOLD` kosong, bukan angka, atau ≤ 0 — fallback ke 20, tidak boleh panic.
- `POST /api/notify/test` saat konfigurasi belum lengkap — balas error jelas, jangan panic atau hang.

---

### Task 1: Config dari environment

**Files:**
- Create: `internal/notify/config.go`
- Test: `internal/notify/config_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Config struct { User, Password, Recipient string; Threshold float64 }`
  - `func ConfigFromEnv() Config`
  - `func (c Config) Configured() bool` — true hanya bila `User` dan `Password` tidak kosong.

- [ ] **Step 1: Write the failing test**

```go
package notify

import "testing"

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("SMTP_USER", "sender@gmail.com")
	t.Setenv("SMTP_PASS", "apppass")
	t.Setenv("NOTIFY_TO", "")
	t.Setenv("NOTIFY_THRESHOLD", "")

	cfg := ConfigFromEnv()
	if cfg.User != "sender@gmail.com" || cfg.Password != "apppass" {
		t.Fatalf("user/password not read: %+v", cfg)
	}
	if cfg.Recipient != "sender@gmail.com" {
		t.Fatalf("recipient should default to user, got %q", cfg.Recipient)
	}
	if cfg.Threshold != 20 {
		t.Fatalf("threshold should default to 20, got %v", cfg.Threshold)
	}
	if !cfg.Configured() {
		t.Fatal("expected configured")
	}
}

func TestConfigFromEnvInvalidThreshold(t *testing.T) {
	t.Setenv("SMTP_USER", "sender@gmail.com")
	t.Setenv("SMTP_PASS", "apppass")
	for _, v := range []string{"abc", "0", "-5"} {
		t.Setenv("NOTIFY_THRESHOLD", v)
		if got := ConfigFromEnv().Threshold; got != 20 {
			t.Fatalf("threshold %q should fallback to 20, got %v", v, got)
		}
	}
}

func TestConfigNotConfiguredWithoutCredentials(t *testing.T) {
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASS", "")
	if ConfigFromEnv().Configured() {
		t.Fatal("expected not configured")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/ -run TestConfig -v`
Expected: FAIL — package/function not defined.

- [ ] **Step 3: Implement `ConfigFromEnv`**

`ConfigFromEnv` reads `os.Getenv`. Empty `NOTIFY_TO` → `User`. Threshold parsed with `strconv.ParseFloat`; parse error, `<= 0`, or `NaN` → `20`. `Configured()` returns `c.User != "" && c.Password != ""`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/notify/ -run TestConfig -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/notify/config.go internal/notify/config_test.go
git commit -m "feat(notify): read email config from environment"
```

---

### Task 2: Pesan email dan pengiriman SMTP

**Files:**
- Create: `internal/notify/smtp.go`
- Test: `internal/notify/smtp_test.go`

**Interfaces:**
- Consumes: `battery.BatteryInfo` (`Percentage float64`, `Status string`).
- Produces:
  - `func buildMessage(from, to, hostname string, info battery.BatteryInfo, now time.Time) []byte`
  - `type Sender func(user, password, to string, msg []byte) error`
  - `func SMTPSend(user, password, to string, msg []byte) error` — cocok dengan tipe `Sender`.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/ -run TestBuildMessage -v`
Expected: FAIL — `buildMessage` undefined.

- [ ] **Step 3: Implement `smtp.go`**

- `buildMessage` menghasilkan email RFC 5322 sederhana: header `From`, `To`, `Subject`, `Date` (`now.Format(time.RFC1123Z)`), `MIME-Version`, `Content-Type: text/plain; charset=UTF-8`; body berisi hostname, persentase (1 desimal), status baterai, dan waktu.
- Subjek: `[SRVWATCH] Baterai <hostname> discharging (<pct>%)`.
- `SMTPSend` memakai `smtp.PlainAuth("", user, password, "smtp.gmail.com")` dan `smtp.SendMail("smtp.gmail.com:587", auth, user, []string{to}, msg)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/notify/ -run TestBuildMessage -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/notify/smtp.go internal/notify/smtp_test.go
git commit -m "feat(notify): build and send alert email via Gmail SMTP"
```

---

### Task 3: State machine `Check` dan `Status`

**Files:**
- Create: `internal/notify/service.go`
- Test: `internal/notify/service_test.go`

**Interfaces:**
- Consumes: `Config`, `Sender`, `buildMessage`, `battery.BatteryInfo`.
- Produces:
  - `type Status struct { Configured bool \`json:"configured"\`; Threshold float64 \`json:"threshold"\`; LastSent string \`json:"last_sent,omitempty"\`; LastError string \`json:"last_error,omitempty"\`; Armed bool \`json:"armed"\` }`
  - `func NewService(cfg Config, send Sender) *Service` — menyimpan hostname dari `os.Hostname()`.
  - `func (s *Service) Check(info battery.BatteryInfo, now time.Time)`
  - `func (s *Service) SendTest() error`
  - `func (s *Service) Status() Status`
  - `func sanitize(msg string, secrets ...string) string` — replaces every non-empty secret with `[redacted]`.

- [ ] **Step 1: Write the failing tests**

```go
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

func TestSendTestNotConfigured(t *testing.T) {
	s := NewService(Config{}, countingSender(new(int), nil))
	if err := s.SendTest(); err == nil {
		t.Fatal("expected error when not configured")
	}
}

func TestSendTestSendsOnce(t *testing.T) {
	calls := 0
	s := NewService(testConfig(), countingSender(&calls, nil))
	if err := s.SendTest(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 test email, got %d", calls)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/ -run 'TestCheck|TestStatus|TestSanitize|TestSendTest' -v`
Expected: FAIL — undefined `NewService`, `Check`, `Status`, `sanitize`.

- [ ] **Step 3: Implement `service.go`**

Rules:
- Struct fields guarded by `sync.Mutex`: `cfg`, `send`, `hostname`, `notified bool`, `lastSent time.Time`, `lastAttempt time.Time`, `lastErr string`.
- `Check(info, now)`:
  - `!cfg.Configured()` → return.
  - Episode reset first: jika `info.Status != "Discharging"` atau `info.Percentage >= cfg.Threshold` → `notified = false`; return.
  - Kirim hanya jika `!notified && info.Percentage > 0` dan (`lastAttempt` zero atau `now.Sub(lastAttempt) >= time.Minute`).
  - Sebelum memanggil `send`, set `lastAttempt = now`. Bangun pesan dengan `buildMessage(cfg.User, cfg.Recipient, hostname, info, now)`.
  - Sukses: `notified = true`, `lastSent = now`, `lastErr = ""`. Gagal: `lastErr = sanitize(err.Error(), cfg.User, cfg.Password, cfg.Recipient)`.
- `SendTest()`: lock, jika `!Configured()` → `errors.New("email notification not configured")`. Salin `cfg`, `send`, `hostname`, unlock, kirim `buildMessage(...)` dengan `time.Now()`, sanitasi error.
- `Status()`: `Configured`, `Threshold`, `LastSent` sebagai `lastSent.Format(time.RFC3339)` bila tidak zero, `LastError`, `Armed = cfg.Configured() && !notified`.
- `LastError` tidak boleh memuat `User`, `Password`, atau `Recipient`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/notify/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/notify/service.go internal/notify/service_test.go
git commit -m "feat(notify): add discharge alert state machine"
```

---

### Task 4: HTTP handlers

**Files:**
- Create: `internal/notify/handler.go`
- Test: `internal/notify/handler_test.go`

**Interfaces:**
- Consumes: `Service.Status() Status`, `Service.SendTest() error`.
- Produces:
  - `func NewHandler(service *Service) *Handler`
  - `func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request)` — hanya `GET`, selain itu `405`.
  - `func (h *Handler) SendTest(w http.ResponseWriter, r *http.Request)` — hanya `POST`, selain itu `405`.

- [ ] **Step 1: Write the failing tests**

```go
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

func TestSendTestHandler(t *testing.T) {
	calls := 0
	h := NewHandler(NewService(testConfig(), countingSender(&calls, nil)))
	rec := httptest.NewRecorder()
	h.SendTest(rec, httptest.NewRequest(http.MethodPost, "/api/notify/test", nil))
	if rec.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status %d calls %d", rec.Code, calls)
	}

	rec = httptest.NewRecorder()
	h.SendTest(rec, httptest.NewRequest(http.MethodGet, "/api/notify/test", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/notify/ -run 'TestGetStatus|TestSendTestHandler' -v`
Expected: FAIL — handler undefined.

- [ ] **Step 3: Implement `handler.go`**

`Content-Type: application/json`, `Access-Control-Allow-Origin: *` mengikuti handler lain. Body sukses test: `{"status":"sent"}`. Error test: `500` dengan `{"error": "<sanitized message>"}` (pesan dari `SendTest` sudah disanitasi; jangan menambahkan alamat).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/notify/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/notify/handler.go internal/notify/handler_test.go
git commit -m "feat(notify): add status and test-email endpoints"
```

---

### Task 5: Wiring backend, dependency, dan `.env.example`

**Files:**
- Modify: `main.go` (import, init, routes, `updateAllData`)
- Modify: `go.mod`, `go.sum`
- Create: `.env.example`
- Create: `.gitignore` (berisi `.env`)

**Interfaces:**
- Consumes: `notify.NewService`, `notify.ConfigFromEnv`, `notify.SMTPSend`, `notify.NewHandler`, `(*notify.Service).Check`.
- Produces: route `GET /api/notify`, `POST /api/notify/test`; `updateAllData` menerima `notifier *notify.Service`.

- [ ] **Step 1: Tambah dependency dan file konfigurasi contoh**

Run: `go get github.com/joho/godotenv@v1.5.1`
Create `.env.example`:
```
SMTP_USER=your-address@gmail.com
SMTP_PASS=xxxx xxxx xxxx xxxx
NOTIFY_TO=your-address@gmail.com
NOTIFY_THRESHOLD=20
```
Create `.gitignore` minimal berisi `.env`.

- [ ] **Step 2: Wire `main.go`**

- `godotenv.Load()` di awal `main()`; error diabaikan (env systemd tetap dipakai).
- `notifyService := notify.NewService(notify.ConfigFromEnv(), notify.SMTPSend)`.
- `notifyHandler := notify.NewHandler(notifyService)`.
- Routes: `http.HandleFunc("/api/notify", notifyHandler.GetStatus)` dan `http.HandleFunc("/api/notify/test", notifyHandler.SendTest)`.
- Tambahkan parameter `notifier *notify.Service` ke `updateAllData` dan panggil `notifier.Check(batteryInfo, time.Now())` tepat setelah `batteryService.GetBatteryInfo()`.
- Tambahkan `/api/notify` ke baris log endpoint.

- [ ] **Step 3: Build dan jalankan seluruh test**

Run: `go build ./... && go test ./...`
Expected: build sukses, semua test PASS.

- [ ] **Step 4: Verifikasi manual tanpa kredensial**

Run: `go run .` di terminal, lalu:
- `curl -s localhost:8081/api/notify` → `{"configured":false,"threshold":20,...}`.
- `curl -s -X POST localhost:8081/api/notify/test` → HTTP 500, pesan error tanpa alamat email.
- Dashboard menampilkan card Email Alerts dengan status NOT CONFIGURED.

- [ ] **Step 5: Commit**

```bash
git add main.go go.mod go.sum .env.example .gitignore
git commit -m "feat: wire battery email alerts into server"
```

---

### Task 6: Card status di dashboard

**Files:**
- Modify: `templates/index.html` (tambah info-card di `.bottom-row`)
- Modify: `static/script.js` (fetch status, handler tombol test)
- Modify: `static/style.css` (gaya tombol test)

**Interfaces:**
- Consumes: `GET /api/notify`, `POST /api/notify/test`.
- Produces: element `#notify-state`, `#notify-rule`, `#notify-last`, `#notify-armed`, `#notify-test`, `#notify-result`.

- [ ] **Step 1: Tambah card di `templates/index.html`**

Setelah card "Network I/O", tambahkan `info-card` dengan judul `Email Alerts` bermuatan:
- `Status` → `#notify-state`
- `Rule` → `#notify-rule`
- `Terakhir` → `#notify-last`
- `Kondisi` → `#notify-armed`
- `<button id="notify-test" class="notify-btn">Kirim test email</button>`
- `<div id="notify-result" class="notify-result"></div>`

Tidak boleh ada alamat email di markup.

- [ ] **Step 2: Tambah gaya tombol di `static/style.css`**

`.notify-btn` mengikuti bahasa visual yang ada (border `var(--border)`, background `var(--surface2)`, teks `var(--text-dim)`, font `Space Mono`, ukuran 11px, padding 8px 12px, radius 6px, cursor pointer, hover border `var(--accent)`). `.notify-result` ukuran 10px, warna `var(--text-dim)`, margin-top 8px.

- [ ] **Step 3: Tambah perilaku di `static/script.js`**

- `async function fetchNotifyStatus()`: `fetch('/api/notify')`, isi:
  - `#notify-state`: `ACTIVE` (warna `var(--accent3)`) atau `NOT CONFIGURED` (`var(--warn)`).
  - `#notify-rule`: `< threshold% saat discharging`.
  - `#notify-last`: `new Date(last_sent).toLocaleString('id-ID')` atau `belum pernah`.
  - `#notify-armed`: `siap` bila `armed`, `menunggu` bila configured tapi tidak armed, `off` bila tidak configured; bila `last_error` ada tampilkan `error` dan teks error (sudah disanitasi server).
- Panggil `fetchNotifyStatus()` sekali saat load lalu `setInterval(fetchNotifyStatus, 2000)`.
- Handler klik `#notify-test`: `POST /api/notify/test`, tampilkan `terkirim` atau pesan error dari JSON di `#notify-result`; nonaktifkan tombol selama request.

- [ ] **Step 4: Verifikasi manual**

Run: `go run .`, buka `http://localhost:8081`:
- Card Email Alerts tampil rapi bersama card lain.
- Status NOT CONFIGURED (tanpa `.env`).
- Tidak ada alamat email di DOM (cek DevTools).
- Dengan `.env` valid: status ACTIVE, tombol test mengirim email dan menampilkan `terkirim`.

- [ ] **Step 5: Commit**

```bash
git add templates/index.html static/script.js static/style.css
git commit -m "feat(ui): show email alert status on dashboard"
```

---

### Task 7: Catatan deployment

**Files:**
- Modify: `readme.txt`

- [ ] **Step 1: Tambah catatan**

Tambahkan ke `readme.txt`: buat `.env` di `/home/fikri/project/monitoring_server/` berisi variabel yang sama dengan `.env.example`, gunakan App Password Gmail 16 digit (2FA aktif), jangan commit `.env`, dan restart service setelah mengubahnya.

- [ ] **Step 2: Commit**

```bash
git add readme.txt
git commit -m "docs: note email alert env setup"
```
