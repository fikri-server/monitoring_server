monitoring server for linux ubuntu server with github action & github runner
test gitacp -m "update"

Email alert saat baterai discharging:
- Buat .env di /home/fikri/project/monitoring_server/ (contoh: .env.example)
- SMTP_USER = alamat Gmail pengirim
- SMTP_PASS = App Password Gmail 16 digit (aktifkan 2FA dulu)
- NOTIFY_TO = penerima (kosong = SMTP_USER), NOTIFY_THRESHOLD = 20
- .env tidak di-commit; restart service setelah mengubahnya
- Status ada di card "Email Alerts" di dashboard (alamat email tidak ditampilkan)
