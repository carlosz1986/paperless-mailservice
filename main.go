package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func main() {
	LoadConfig()

	PrintRules()

	// if runEveryXMinute is set, a ticker executes the logic over and over again, otherwise the logic is executed once
	rand.Seed(time.Now().UnixNano())

	if err := processJob(); err != nil {
		log.Fatalf("error Process Job: %v", err)
	}

	if Config.RunEveryXMinute == -1 {
		return
	}

	ticker := time.NewTicker(time.Duration(Config.RunEveryXMinute) * time.Minute)
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

	correspondents, err := getCorrespondents()
	if err != nil {
		return fmt.Errorf("error getting correspondents: %v", err)
	}

	documentTypes, err := getDocumentTypes()
	if err != nil {
		return fmt.Errorf("error getting document types: %v", err)
	}

	storagePaths, err := getStoragePaths()
	if err != nil {
		return fmt.Errorf("error getting storage pathes: %v", err)
	}

	users, err := getUsers()
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	processedTag := getTagByName(tags, Config.Paperless.ProcessedTagName)
	if processedTag == nil {
		return fmt.Errorf("error finding processedTagName:%s in list from server", Config.Paperless.ProcessedTagName)
	}

	searchTag := getTagByName(tags, Config.Paperless.AddQueueTagName)
	if searchTag == nil {
		return fmt.Errorf("error finding searchTagName:%s in list from from server", Config.Paperless.AddQueueTagName, err)
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
		// check if rule for document exists, and process doc
		// all tags of a rule need to be available for the doc
		atLeastOneRuleMatches := false

		for _, rule := range Config.Paperless.Rules {
			docMatchesRule := false
			for _, ruleTag := range rule.Tags {
				foundDocTag := false
				for _, id := range doc.TagIDs {
					found := getTagByID(tags, id)
					if found == nil {
						return fmt.Errorf("Tag %d is not available in tags list", id)
					}

					if ruleTag == found.Name {
						foundDocTag = true
					}
				}
				if foundDocTag {
					docMatchesRule = true
				} else {
					docMatchesRule = false
					break
				}
			}
			if !docMatchesRule {
				// if the rule does not match, try next rule
				continue
			}
			// found a rule that matches, start processing
			log.Printf("found Rule: %s, that matches Tag(s) (%s) in document: '%s' (%d)", rule.Name, strings.Join(rule.Tags, ","), doc.getFileName(), doc.ID)

			user := getUserByID(users, doc.OwnerId)
			if user == nil {
				log.Printf("warning: could not find user for doc with id=%d, placeholders won't work", doc.ID)
			}

			correspondent := getCorrespondentByID(correspondents, doc.CorrespondentId)
			if correspondent == nil {
				log.Printf("warning: could not find a correspondent for doc with id=%d, placeholders won't work", doc.ID)
			}

			documentType := getDocumentTypeByID(documentTypes, doc.DocumentTypeId)
			if documentType == nil {
				log.Printf("warning: could not find a document type for doc with id=%d, placeholders won't work", doc.ID)
			}

			storagePath := getStoragePathByID(storagePaths, doc.StoragePath)
			if storagePath == nil {
				log.Printf("warning: could not find a storage path for doc with id=%d, placeholders won't work", doc.ID)
			}

			mailHeader := prepareMail(Config.Email.MailHeader, rule.MailHeader, user, correspondent, documentType, storagePath, &doc)
			mailBody := prepareMail(Config.Email.MailBody, rule.MailBody, user, correspondent, documentType, storagePath, &doc)

			if err := SendProcessDoc(doc, processedTag, mailHeader, mailBody, rule.ReceiverAddress); err != nil {
				log.Printf("error processing Doc: %v", err)
				continue
			}

			log.Printf("document '%s' (%d) succesfully sent to '%s'", doc.getFileName(), doc.ID, rule.ReceiverAddress)
			atLeastOneRuleMatches = true
			//goto nextDoc
		}
		if !atLeastOneRuleMatches {
			log.Printf("document '%s' (%d) marked for processing, but no Ruleset matches the tags ...", doc.getFileName(), doc.ID)
		}
		//nextDoc:
	}

	return nil
}

func prepareMail(str, ruleStr string, user *User, correspondent *Correspondent, documenType *DocumentType, storagePath *StoragePath, document *Document) string {
	// use the header,body string from rule if set
	if ruleStr != "" {
		str = ruleStr
	}

	if user != nil {
		str = strings.ReplaceAll(str, "%user_name%", user.Username)
		str = strings.ReplaceAll(str, "%user_id%", strconv.Itoa(user.ID))
		str = strings.ReplaceAll(str, "%user_email%", user.Email)
		str = strings.ReplaceAll(str, "%first_name%", user.FirstName)
		str = strings.ReplaceAll(str, "%last_name%", user.LastName)
	}

	if correspondent != nil {
		str = strings.ReplaceAll(str, "%correspondent_name%", correspondent.Name)
		str = strings.ReplaceAll(str, "%correspondent_id%", strconv.Itoa(correspondent.ID))

	}

	if documenType != nil {
		str = strings.ReplaceAll(str, "%document_type_name%", documenType.Name)
		str = strings.ReplaceAll(str, "%document_type_id%", strconv.Itoa(documenType.ID))

	}

	if storagePath != nil {
		str = strings.ReplaceAll(str, "%storage_path_name%", storagePath.Name)
		str = strings.ReplaceAll(str, "%storage_path_id%", strconv.Itoa(storagePath.ID))
		str = strings.ReplaceAll(str, "%storage_path%", storagePath.Path)
	}

	str = strings.ReplaceAll(str, "%document_id%", strconv.Itoa(document.ID))
	str = strings.ReplaceAll(str, "%document_url%", document.getDocumentURL())
	str = strings.ReplaceAll(str, "%document_title%", document.Title)
	str = strings.ReplaceAll(str, "%document_file_name%", document.getFileName())
	str = strings.ReplaceAll(str, "%document_created_at%", document.CreatedAt)
	str = strings.ReplaceAll(str, "%document_modified_at%", document.ModifiedAt)

	return str
}

func SendProcessDoc(doc Document, processedTag *Tag, mailHeader, mailBody, receiverAddress string) error {
	// download document
	bytes, err := downloadDocumentBinary(doc)
	if err != nil {
		return fmt.Errorf("failed to download document: '%s' (%d): %v", doc.getFileName(), doc.ID, err)
	}

	log.Printf("downloaded document: '%s' (%d)", doc.getFileName(), doc.ID)

	// found right rule, send it
	err = SendEmailWithPDFBinaryAttachment(Config.Email.SMTPServer,
		Config.Email.SMTPPort,
		Config.Email.SMTPConnectionType,
		Config.Email.SMTPAddress,
		Config.Email.SMTPUser,
		Config.Email.SMTPPassword,
		receiverAddress,
		mailHeader,
		mailBody,
		doc.getFileName(),
		bytes)

	if err != nil {
		return fmt.Errorf("error sending email: %v", err)
	}

	err = addTagToDocument(doc, *processedTag)
	if err != nil {
		return fmt.Errorf("could not add Tag for document '%s' (%d): %v", doc.getFileName(), doc.ID, err)
	}
	return nil
}
