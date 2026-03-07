package common

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

func SendEmail(subject string, receiver string, content string) error {
	if SMTPServer == "" || SMTPAccount == "" {
		return fmt.Errorf("SMTP not configured")
	}

	from := SMTPFrom
	if from == "" {
		from = SMTPAccount
	}

	msg := []byte("To: " + receiver + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + content + "\r\n")

	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	auth := smtp.PlainAuth("", SMTPAccount, SMTPPassword, SMTPServer)

	if SMTPSSLEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         SMTPServer,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			return err
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(from); err != nil {
			return err
		}
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
		w, err := client.Data()
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
		return client.Quit()
	}

	return smtp.SendMail(addr, auth, from, []string{receiver}, msg)
}
