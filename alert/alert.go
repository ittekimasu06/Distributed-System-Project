package alert

import (
    "fmt"
    "log"
    "net/smtp"
    "time"
)

type AlertSystem struct {
    smtpHost      string
    smtpPort      string
    smtpUsername  string
    smtpPassword  string
    alertEmail    string
    threshold     float64
    lastAlert     time.Time
    alertDuration time.Duration
}

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

func (a *AlertSystem) CheckAndSendAlert(cpuPercent float64) error {
    if a.smtpHost == "" {
        log.Printf("SMTP disabled, no email sent")
        return nil
    }
    if cpuPercent > a.threshold && time.Since(a.lastAlert) >= a.alertDuration {
        log.Printf("CPU usage %.2f%% above threshold %.2f%%, sending alert", cpuPercent, a.threshold)
        err := a.sendEmail(cpuPercent)
        if err != nil {
            log.Printf("Failed to send email: %v", err)
            return err
        }
        a.lastAlert = time.Now()
        log.Printf("Sent email alert for high CPU usage: %.2f%%", cpuPercent)
    }
    return nil
}

func (a *AlertSystem) sendEmail(cpuPercent float64) error {
    from := a.smtpUsername
    subject := "High CPU Usage Alert"
    body := fmt.Sprintf("CPU usage on localhost has exceeded %.2f%%. Current usage: %.2f%%", a.threshold, cpuPercent)
    msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", from, a.alertEmail, subject, body)
    auth := smtp.PlainAuth("", a.smtpUsername, a.smtpPassword, a.smtpHost)
    err := smtp.SendMail(a.smtpHost+":"+a.smtpPort, auth, from, []string{a.alertEmail}, []byte(msg))
    if err != nil {
        return fmt.Errorf("failed to send email: %v", err)
    }
    return nil
}