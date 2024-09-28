package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/quotedprintable"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

type Document struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	FileName string `json:"archived_file_name"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	// if runEveryXMinute is set, a ticker executes the logic over and over again, otherwise the logic is executed once
	rand.Seed(time.Now().UnixNano())
	runEveryXMinute, err := strconv.Atoi(os.Getenv("runEveryXMinute"))
	if err != nil {
		log.Fatalf("runEveryXMinute Environment Variable is not a valid Number")
	}

	if err := processJob(); err != nil {
		log.Fatalf("error Process Job: %v", err)
	}

	if runEveryXMinute == -1 {
		return
	}

	ticker := time.NewTicker(time.Duration(runEveryXMinute) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := processJob(); err != nil {
			log.Fatalf("error Process Job: %v", err)
		}
	}
}

func processJob() error {
	tags, err := getTags()
	if err != nil {
		return fmt.Errorf("error getting tags: %v", err)
	}

	processedTag, err := getTagByName(tags, os.Getenv("processedTagName"))
	if err != nil {
		return fmt.Errorf("error loading processedTagName:%s from server: %v", os.Getenv("processedTagName"), err)
	}

	searchTag, err := getTagByName(tags, os.Getenv("searchTagName"))
	if err != nil {
		return fmt.Errorf("error loading searchTagName:%s from server: %v", os.Getenv("searchTagName"), err)
	}

	documents, err := getDocumentsByTag(*searchTag, *processedTag)
	if err != nil {
		fmt.Errorf("error getting documents with tag: %v", err)
		return err
	}

	if len(documents) == 0 {
		log.Println("no documents found to process")
		return nil
	}

	for _, doc := range documents {
		bytes, err := downloadDocumentBinary(doc)
		if err != nil {
			fmt.Errorf("failed to download document ID %d: %v", doc.ID, err)
		} else {
			log.Printf("downloaded document: %s", doc.FileName)

			err = SendEmailWithPDFBinaryAttachment(os.Getenv("smtpServer"), os.Getenv("smtpPort"), os.Getenv("smtpConnectionType"),
				os.Getenv("smtpEmail"), os.Getenv("smtpUser"), os.Getenv("smtpPassword"), os.Getenv("receiverEmail"),
				os.Getenv("mailHeader"), os.Getenv("mailBody"), doc.FileName, bytes)

			if err != nil {
				return fmt.Errorf("error sending email: %v", err)
			}

			err = addTagToDocument(doc, *processedTag)
			if err != nil {
				return fmt.Errorf("coold not add Tag: %v", err)
			}
			log.Printf("document '%s' succesfully sent & processed", doc.FileName)
		}
	}
	return nil
}

func getTags() ([]Tag, error) {
	var tags []Tag
	page := 1

	for {
		url := fmt.Sprintf("%stags/?page=%d", os.Getenv("paperlessInstanceURL"), page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", os.Getenv("paperlessInstanceToken")))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch tags: %s", resp.Status)
		}

		var result struct {
			Results []Tag  `json:"results"`
			Next    string `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		for _, tag := range result.Results {
			tags = append(tags, tag)
		}

		if result.Next == "" {
			break
		}
		page++
	}

	return tags, nil
}

func getDocumentsByTag(tag Tag, processedTag Tag) ([]Document, error) {
	var documents []Document
	page := 1

	for {
		url := fmt.Sprintf("%sdocuments/?page=%d&tags__id__all=%d&tags__id__none=%d", os.Getenv("paperlessInstanceURL"), page, tag.ID, processedTag.ID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", os.Getenv("paperlessInstanceToken")))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch documents: %s", resp.Status)
		}

		var result struct {
			Results []Document `json:"results"`
			Next    string     `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		documents = append(documents, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return documents, nil
}

func addTagToDocument(document Document, tag Tag) error {
	url := fmt.Sprintf("%sdocuments/bulk_edit/", os.Getenv("paperlessInstanceURL"))

	type payload struct {
		Documents  []int          `json:"documents"`
		Method     string         `json:"method"`
		Parameters map[string]int `json:"parameters"`
	}
	p, b := payload{
		Documents:  []int{document.ID},
		Method:     "add_tag",
		Parameters: map[string]int{"tag": tag.ID},
	}, new(bytes.Buffer)

	err := json.NewEncoder(b).Encode(p)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", os.Getenv("paperlessInstanceToken")))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bulk Edit failed, Unexpected server status code: %d", resp.StatusCode)
	}

	return nil
}

func downloadDocumentBinary(doc Document) ([]byte, error) {
	url := fmt.Sprintf("%sdocuments/%d/download/", os.Getenv("paperlessInstanceURL"), doc.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", os.Getenv("paperlessInstanceToken")))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download document ID %d: %s", doc.ID, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getTagByName(tags []Tag, name string) (*Tag, error) {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag, nil
		}
	}
	return nil, fmt.Errorf("Tag %s is not available in server tags list", name)
}

func SendEmailWithPDFBinaryAttachment(smtpHost, smtpPort, connectionType, sender, user, password, recipient, subject, body, filename string, attachment []byte) error {
	// Create the email header
	header := make(map[string]string)
	header["From"] = sender
	header["To"] = recipient
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = `multipart/mixed; boundary="BOUNDARY"`
	header["Date"] = time.Now().Format(time.RFC1123Z)

	var emailBuf bytes.Buffer
	for k, v := range header {
		emailBuf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	emailBuf.WriteString("\r\n--BOUNDARY\r\n")

	// Create the body part
	emailBuf.WriteString(`Content-Type: text/plain; charset="utf-8"` + "\r\n")
	emailBuf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	// write the email body to the buffer
	qp := quotedprintable.NewWriter(&emailBuf)
	_, err := qp.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("unable to write body: %v", err)
	}
	qp.Close()

	emailBuf.WriteString("\r\n--BOUNDARY\r\n")

	// Create the attachment part
	emailBuf.WriteString(fmt.Sprintf(`Content-Type: application/pdf; name="%s"`+"\r\n", filename))
	emailBuf.WriteString("Content-Transfer-Encoding: base64\r\n")
	emailBuf.WriteString(fmt.Sprintf(`Content-Disposition: attachment; filename="%s"`+"\r\n\r\n", filename))

	b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
	base64.StdEncoding.Encode(b, attachment)

	if err := addLinesSplittedToBuffer(b, &emailBuf); err != nil {
		return fmt.Errorf("failed to add line separators to BinaryFile: %v", err)
	}

	emailBuf.WriteString("\r\n--BOUNDARY--")

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

	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("failed to set mail receiver: %v", err)
	}

	// Get the data writer to send the email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get writer: %v", err)
	}
	defer w.Close()

	//log.Print(emailBuf.String())
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
