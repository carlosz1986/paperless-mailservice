package main

import (
	"os"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/quotedprintable"
	"net/http"
	"net/smtp"
	"time"
	"math/rand"
	"strconv"
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
        log.Fatalf("Error Process Job: %v", err)
    }
	
	if runEveryXMinute == -1 {
		return
	}

	ticker := time.NewTicker(time.Duration(runEveryXMinute) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
        if err := processJob(); err != nil {
            log.Fatalf("Error Process Job: %v", err)
        }
    }
}

func processJob() error {
	tags, err := getTags()
	if err != nil {
		return fmt.Errorf("Error getting tags: %v", err)
	}

	processedTag, err := getTagByName(tags, os.Getenv("processedTagName"))
	if err != nil {
		fmt.Errorf("Error loading processedTagName:%s from server: %v", os.Getenv("processedTagName"), err)
	}

	searchTag, err := getTagByName(tags, os.Getenv("searchTagName"))
	if err != nil {
		fmt.Errorf("Error loading searchTagName:%s from server: %v", os.Getenv("searchTagName"), err)
	}

	documents, err := getDocumentsByTag(*searchTag, *processedTag)
	if err != nil {
		fmt.Errorf("Error getting documents with tag: %v", err)
		return err
	}

	if len(documents) == 0 {
		log.Println("No documents found to process")
		return nil
	}

	for _, doc := range documents {
		bytes, err := downloadDocumentBinary(doc)
		if err != nil {
			fmt.Errorf("Failed to download document ID %d: %v", doc.ID, err)
		} else {
			log.Printf("Downloaded document: %s", doc.FileName)

			err = SendEmailWithPDFBinaryAttachment(os.Getenv("smtpServer"), os.Getenv("smtpPort"),
				os.Getenv("smtpEmail"), os.Getenv("smtpPassword"), os.Getenv("receiverEmail"),
				os.Getenv("mailHeader"), os.Getenv("mailBody"), doc.FileName, bytes)
				
			if err != nil {
				return fmt.Errorf("Error sending email: %v", err)
			}

			err = addTagToDocument(doc, *processedTag)
			if err != nil {
				return fmt.Errorf("Coold not add Tag: %v", err)
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
		return fmt.Errorf("Failed to marshal payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return fmt.Errorf("Failed to create request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", os.Getenv("paperlessInstanceToken")))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bulk Edit failed, Unexpected server status code: %d", resp.StatusCode)
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

func SendEmailWithPDFBinaryAttachment(smtpHost, smtpPort, sender, password, recipient, subject, body, filename string, attachment []byte) error {
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

	b64 := base64.NewEncoder(base64.StdEncoding, &emailBuf)
	_, err = b64.Write(attachment)
	if err != nil {
		return fmt.Errorf("unable to write file content to base64: %v", err)
	}
	b64.Close()
	emailBuf.WriteString("\r\n--BOUNDARY--")

	auth := smtp.PlainAuth("", sender, password, smtpHost)
	err = smtp.SendMail(fmt.Sprintf("%s:%s", smtpHost, smtpPort), auth, sender, []string{recipient}, emailBuf.Bytes())

	if err != nil {
		return fmt.Errorf("unable to send email: %v", err)
	}

	return nil
}
