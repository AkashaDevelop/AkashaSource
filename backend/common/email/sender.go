package email

import (
	"STfreApi/common"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

func SendEmail(to string, subject string, body string) error {
	if common.SMTPServer == "" {
		return fmt.Errorf("SMTP server not configured")
	}

	addr := fmt.Sprintf("%s:%d", common.SMTPServer, common.SMTPPort)
	auth := smtp.PlainAuth("", common.SMTPAccount, common.SMTPPassword, common.SMTPServer)

	msg := []byte("To: " + to + "\r\n" +
		"From: " + fmt.Sprintf("%s <%s>", common.SystemName, common.SMTPFrom) + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		body)

	if common.SMTPSSLEnabled {
		// SSL/TLS connection (usually port 465)
		return sendMailTLS(addr, auth, common.SMTPFrom, []string{to}, msg)
	} else {
		// STARTTLS or Plain (usually port 587 or 25)
		return smtp.SendMail(addr, auth, common.SMTPFrom, []string{to}, msg)
	}
}

func sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create a TLS connection
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true, // For self-signed certs
		ServerName:         strings.Split(addr, ":")[0],
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer c.Quit()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}
