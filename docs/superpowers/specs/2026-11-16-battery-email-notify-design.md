# Notifikasi Email Gmail saat Baterai Discharged

Tanggal: 2026-11-16

## Intent

Server monitoring ini berjalan di Ubuntu (laptop) dan sudah membaca status baterai
dari `/sys/class/power_supply` tiap 2 detik. Pemilik ingin diberi tahu lewat email
kalau baterai sedang **discharging dan di bawah 20%** — kemungkinan karena listrik
mati atau charger lepas — supaya sempat bertindak sebelum server shutdown.

Email dikirim dari dan ke Gmail pemilik sendiri (SMTP Gmail + App Password).

## Goals

- Kirim email otomatis sekali per episode discharge ketika status `Discharging`
  dan persentase < threshold.
- Konfigurasi sepenuhnya dari `.env`/environment server.
- Dashboard menampilkan status notifikasi tanpa membocorkan alamat email atau kredensial.
- Tombol "Kirim test email" untuk memverifikasi kredensial kapan saja.

## Non-goals

- Tidak ada UI untuk mengubah setting email (alamat, password, threshold).
- Tidak ada penyimpanan kredensial di file config atau database.
- Tidak ada dukungan penyedia email selain Gmail SMTP.
- Tidak ada autentikasi dashboard / hardening tambahan untuk port 8081.
- Tidak ada notifikasi ke Telegram/WhatsApp/SMS.

## Anti-goals

- Spam email: tidak boleh lebih dari satu email per episode discharge.
- Membocorkan App Password atau alamat email lewat API, log, atau HTML.

## Constraints

- Bahasa Go 1.22, stdlib `net/smtp` untuk pengiriman.
- Perlu `github.com/joho/godotenv` untuk membaca `.env` saat startup; env systemd
  tetap dihormati (godotenv tidak menimpa env yang sudah ada).
- Deployment via GitHub self-hosted runner ke `/home/fikri/project/monitoring_server`;
  `.env` harus dibuat manual di server dan tidak masuk git.
- Gmail memerlukan App Password 16 digit (2FA aktif), bukan password akun.

## Approaches

1. **Konfigurasi dari `.env`, UI read-only (dipilih).**
   Kredensial hanya dibaca dari environment. Dashboard hanya menampilkan status,
   tanpa alamat email. Paling aman: tidak ada jalur untuk mengubah atau membaca
   kredensial dari browser.

2. **Setting bisa diubah dari UI, disimpan ke file config.**
   Lebih fleksibel, tapi App Password tersimpan di disk dan siapa pun yang bisa
   membuka dashboard bisa menggantinya. Ditolak oleh pemilik.

3. **Tanpa UI sama sekali, hanya log server.**
   Konfigurasi tetap di `.env`, tetapi tidak ada umpan balik di dashboard.
   Ditolak karena pemilik ingin melihat status dan tombol test di UI.

## Design

### Konfigurasi (environment / `.env`)

| Variable | Wajib | Default | Keterangan |
|---|---|---|---|
| `SMTP_USER` | ya | — | alamat Gmail pengirim |
| `SMTP_PASS` | ya | — | App Password Gmail 16 digit |
| `NOTIFY_TO` | tidak | `SMTP_USER` | alamat penerima |
| `NOTIFY_THRESHOLD` | tidak | `20` | ambang persen |

Bila `SMTP_USER` atau `SMTP_PASS` kosong, fitur email dianggap **not configured**;
tidak ada percobaan koneksi SMTP.

### Package `internal/notify`

- `Config` dibaca dari environment saat startup.
- `Notifier` menyimpan state di memori (tidak dipersist):
  `notified bool`, `lastSent time.Time`, `lastErr string`, `lastAttempt time.Time`.
- `Check(battery.BatteryInfo)` dipanggil dari loop 2 detik di `main.go`:
  - jika tidak configured → tidak melakukan apa-apa;
  - jika `Status == "Discharging"` dan `Percentage < threshold` dan `!notified`
    dan sudah lewat cooldown 1 menit sejak percobaan terakhir → kirim email;
  - saat berhasil: `notified = true`, catat `lastSent`;
  - saat gagal: simpan `lastErr`, ulangi setelah cooldown 1 menit selama kondisi berlaku;
  - jika status bukan `Discharging` atau `Percentage >= threshold` → `notified = false`
    (episode baru akan mengirim email lagi).
- `SendTest()` mengirim email percobaan; error dikembalikan ke caller.
- Pengiriman: `smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, msg)`
  dengan STARTTLS dan `smtp.PlainAuth`. Isi email: hostname, persen baterai,
  status, waktu server.
- Alamat email tidak pernah ditulis ke log. Error SMTP yang memuat alamat
  di-sanitasi sebelum disimpan/disajikan.

### API

- `GET /api/notify` → JSON status:
  ```json
  {
    "configured": true,
    "threshold": 20,
    "last_sent": "2026-11-16T14:32:00+07:00",
    "last_error": "",
    "armed": true
  }
  ```
  Tidak memuat `SMTP_USER`, `SMTP_PASS`, maupun `NOTIFY_TO`.
- `POST /api/notify/test` → kirim email test; balas 200 atau 500 dengan pesan error
  yang sudah disanitasi.
- Tidak ada endpoint untuk mengubah konfigurasi.

### Dashboard

Card "Email Alerts" di `bottom-row` `templates/index.html`, mengikuti gaya card yang ada:

```
┌─ Email Alerts ─────────────────────┐
│ ACTIVE · < 20% saat discharging    │
│ Terakhir : 16 Nov 14:32             │
│ Status   : ● siap                   │
│ [ Kirim test email ]                │
└────────────────────────────────────┘
```

- `static/script.js` mengambil `/api/notify` tiap 2 detik bersama `/api/monitor`
  (fetch terpisah), menampilkan badge ACTIVE / NOT CONFIGURED, threshold, waktu
  terakhir kirim, status siap/menunggu/error.
- Tombol test memanggil `POST /api/notify/test` dan menampilkan hasil singkat.
- Tidak ada alamat email di DOM.

### Wiring `main.go`

- Panggil `godotenv.Load()` saat startup (abaikan error bila `.env` tidak ada).
- Buat `notify.NewService()`; daftarkan `GET /api/notify` dan `POST /api/notify/test`.
- Teruskan notifier ke `updateAllData` dan panggil `Check(batteryInfo)` setiap iterasi.

## Yang diserahkan ke builder

- Format/isi email final (subjek, isi teks) — cukup informatif dan jelas.
- Cara sanitasi pesan error SMTP agar alamat email tidak bocor.
- Detail tampilan card mengikuti CSS yang sudah ada; tidak ada perubahan design language.
