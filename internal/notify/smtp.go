package notify

import (
	"fmt"
	"net/smtp"
	"time"

	"monitoring_server/internal/battery"
)

const smtpServer = "smtp.gmail.com:587"

// Sender matches net/smtp's SendMail shape so the service can be tested
// without a live SMTP server.
type Sender func(user, password, to string, msg []byte) error

func buildMessage(from, to, hostname string, info battery.BatteryInfo, now time.Time) []byte {
	subject := fmt.Sprintf("[SRVWATCH] Baterai %s discharging (%.1f%%)", hostname, info.Percentage)
	body := fmt.Sprintf(
		"Baterai server %s sedang discharging.\n\n"+
			"Persentase : %.1f%%\n"+
			"Status     : %s\n"+
			"Waktu      : %s\n\n"+
			"Segera hubungkan charger sebelum server mati.\n",
		hostname, info.Percentage, info.Status, now.Format(time.RFC3339),
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, now.Format(time.RFC1123Z), body,
	)
	return []byte(msg)
}

func SMTPSend(user, password, to string, msg []byte) error {
	auth := smtp.PlainAuth("", user, password, "smtp.gmail.com")
	return smtp.SendMail(smtpServer, auth, user, []string{to}, msg)
}
