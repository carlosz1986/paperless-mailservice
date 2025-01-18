package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/quotedprintable"
	"net/smtp"
	"strings"
	"time"
)

func toQuotedPrintable(s string) (string, error) {
	var ac bytes.Buffer
	w := quotedprintable.NewWriter(&ac)
	defer w.Close()

	if _, err := w.Write([]byte(s)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return ac.String(), nil
}

// createSubject adds correct line breaks and encodes to quoted printable
func createSubject(s string) (string, error) {
	partP, err := toQuotedPrintable(s)
	if err != nil {
		return "", err
	}
	lines := strings.Split(partP, "\r\n")

	var out string

	for i, part := range lines {
		if i > 0 {
			// seperate every chunk with space
			out += " "
		}
		out += fmt.Sprintf("%s%s%s", "=?UTF-8?Q?", part, "?=")

		// last line of the subject has no line break, as this is added to each header later anyway
		if i < len(lines)-1 {
			out += "\r\n"
		}
	}
	return out, nil
}

// SendEmailWithPDFBinaryAttachment sends email with pdf Binary attachment
func SendEmailWithPDFBinaryAttachment(smtpHost, smtpPort, connectionType, sender, user, password, subject, body, filename string, bCCAddresses, recipients []string, attachment []byte) error {
	// create quoted printable with correct line breaks for the subject
	subjectP, err := createSubject(subject)
	if err != nil {
		return err
	}

	// body to quoted printable
	bodyP, err := toQuotedPrintable(body)
	if err != nil {
		return err
	}

	// Create the email header
	header := make(map[string]string)
	header["From"] = sender
	header["To"] = strings.Join(recipients, ",")
	header["Subject"] = subjectP
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = `multipart/mixed; boundary="BOUNDARY"`
	header["Date"] = time.Now().Format(time.RFC1123Z)

	var emailBuf bytes.Buffer
	for k, v := range header {
		emailBuf.WriteString(fmt.Sprintf("%s: %s%s", k, v, "\r\n"))
	}
	emailBuf.WriteString(fmt.Sprintf(`%s--BOUNDARY%s`, "\r\n", "\r\n"))

	// Create the body part
	emailBuf.WriteString(fmt.Sprintf(`Content-Type: text/html; charset=UTF-8%s`, "\r\n"))
	emailBuf.WriteString(fmt.Sprintf(`Content-Transfer-Encoding: quoted-printable%s`, "\r\n\r\n"))

	// write the email body to the buffer
	emailBuf.WriteString(bodyP)

	emailBuf.WriteString(fmt.Sprintf(`%s--BOUNDARY%s`, "\r\n", "\r\n"))

	// Create the attachment part
	emailBuf.WriteString(fmt.Sprintf(`Content-Type: application/pdf; name="%s"%s`, filename, "\r\n"))
	emailBuf.WriteString(fmt.Sprintf(`Content-Transfer-Encoding: base64%s`, "\r\n"))
	emailBuf.WriteString(fmt.Sprintf(`Content-Disposition: attachment; filename="%s"%s`, filename, "\r\n\r\n"))

	b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
	base64.StdEncoding.Encode(b, attachment)

	if err := addLinesSplittedToBuffer(b, &emailBuf); err != nil {
		return fmt.Errorf("failed to add line separators to BinaryFile: %v", err)
	}

	emailBuf.WriteString(fmt.Sprintf(`%s--BOUNDARY--`, "\r\n"))

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	auth := smtp.PlainAuth("", user, password, smtpHost)

	var client *smtp.Client

	if connectionType == "tls" {

		// Create an SSL/TLS connection
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         smtpHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			if err.Error() == "tls: first record does not look like a TLS handshake" {
				return fmt.Errorf("failed to dial TLS: %v - Try to change smtpConnectionType Config", err)
			}
			return fmt.Errorf("failed to dial TLS: %v", err)
		}

		// Create new client using the SSL connection
		client, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Close()

		// Authenticate
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %v", err)
		}

	} else if connectionType == "starttls" {
		// Handle TLS/STARTTLS (port 587)
		client, err = smtp.Dial(addr)
		if err != nil {
			if err.Error() == "EOF" {
				return fmt.Errorf("failed to dial: %v - Try to change smtpConnectionType Config", err)
			}
			return fmt.Errorf("failed to dial: %v", err)
		}
		defer client.Close()

		// TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         smtpHost,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %v", err)
		}

		// Authenticate
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("failed toauthenticate: %v", err)
		}
	} else {
		return fmt.Errorf("given SMTP connection Type invalid")
	}

	// Set the sender and recipient
	if err := client.Mail(sender); err != nil {
		return fmt.Errorf("failed to set mail sender: %v", err)
	}

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set mail receiver: %v", err)
		}
	}
	// bcc receivers are added to the email, but not shown in the header
	for _, recipient := range bCCAddresses {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set mail BCC receiver: %v", err)
		}
	}

	// Get the data writer to send the email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer: %v", err)
	}
	defer w.Close()

	// Write the message to the data writer
	_, err = w.Write(emailBuf.Bytes())
	if err != nil {
		return fmt.Errorf("unable to send email: %v", err)
	}

	client.Quit()

	return nil
}

func addLinesSplittedToBuffer(b []byte, emailBuf *bytes.Buffer) error {
	// limit the lines to up to 76 chars
	for i, l := 0, len(b); i < l; i++ {
		if err := emailBuf.WriteByte(b[i]); err != nil {
			return err
		}
		if (i+1)%76 == 0 {
			if _, err := emailBuf.WriteString("\r\n"); err != nil {
				return err
			}

		}
	}
	return nil
}
