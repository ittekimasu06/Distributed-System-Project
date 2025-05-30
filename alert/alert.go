package alert

import (
	"fmt"
	"log"
	"net/smtp"
	"time"
)

type AlertSystem struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	alertEmail   string
	threshold    float64
	highCPUSince time.Time
	alertDuration time.Duration
}

// tạo alertsystem từ struct
func NewAlertSystem(smtpHost, smtpPort, smtpUsername, smtpPassword, alertEmail string, threshold float64) *AlertSystem {
	return &AlertSystem{
		smtpHost:      smtpHost,
		smtpPort:      smtpPort,
		smtpUsername:  smtpUsername,
		smtpPassword:  smtpPassword,
		alertEmail:    alertEmail,
		threshold:     threshold,
		alertDuration: 30 * time.Second,
	}
}

// checks mức sdung CPU và gửi alert nếu cần
func (a *AlertSystem) CheckAndSendAlert(cpuPercent float64) error {
	if a.smtpHost == "" {
		return nil // disabled
	}

	if cpuPercent > a.threshold {
		if a.highCPUSince.IsZero() {
			a.highCPUSince = time.Now()
		} else if time.Since(a.highCPUSince) >= a.alertDuration {
			err := a.sendEmail(cpuPercent)
			if err != nil {
				return err
			}
			log.Printf("Sent email alert for high CPU usage: %.2f%%", cpuPercent)
			a.highCPUSince = time.Now() // reset
		}
	} else {
		a.highCPUSince = time.Time{} // reset nếu dưới ngưỡng
	}
	return nil
}

// gửi email
func (a *AlertSystem) sendEmail(cpuPercent float64) error {
	from := a.smtpUsername
	subject := "High CPU Usage Alert"
	body := fmt.Sprintf("CPU usage on localhost has exceeded %.2f%% for over 30 seconds. Current usage: %.2f%%", a.threshold, cpuPercent)
	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", from, a.alertEmail, subject, body)

	auth := smtp.PlainAuth("", a.smtpUsername, a.smtpPassword, a.smtpHost)
	return smtp.SendMail(a.smtpHost+":"+a.smtpPort, auth, from, []string{a.alertEmail}, []byte(msg))
}