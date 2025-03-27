package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Doucment represents a paperless Document
type Document struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	FileName         string `json:"archived_file_name"`
	OriginalFileName string `json:"original_file_name"`
	TagIDs           []int  `json:"tags"`
	CreatedAt        string `json:"created"`
	ModifiedAt       string `json:"modified"`
	CorrespondentId  int    `json:"correspondent"`
	DocumentTypeId   int    `json:"document_type"`
	StoragePath      int    `json:"storage_path"`
	OwnerId          int    `json:"owner"`
	MediaFilename    string `json:"media_filename"`
	Size             int    `json:"original_size"`
}

// getFileName returns the archived filename. For encrypted files it uses the original name.
// If you are using a custom file format and the config variable "UseCustomFilenameFormat" is set to true, it returns the custom filename.
func (d *Document) getFileName() string {
	if Config.Paperless.UseCustomFilenameFormat && d.MediaFilename != "" {
		return d.MediaFilename
	}

	if d.FileName != "" {
		return d.FileName
	}
	return d.OriginalFileName
}

// getDocumentURL returns the Url to the document inside Paperless
func (d *Document) getDocumentURL() string {
	return fmt.Sprintf("%sdocuments/%d/details", Config.Paperless.InstanceURL, d.ID)
}

// Tag represents a paperless Tag
type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Correspondent represents a paperless correspondent
type Correspondent struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DocumentType represents a paperless documentType
type DocumentType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// StoragePath represents a paperless storagePath
type StoragePath struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// User represents a paperless user e.g. owner
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func getRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", Config.Paperless.InstanceToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data: %s", resp.Status)
	}
	return resp, nil
}

func getCorrespondents() ([]Correspondent, error) {
	var correspondents []Correspondent
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/correspondents/?page=%d", Config.Paperless.InstanceURL, page))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch correspondents: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []Correspondent `json:"results"`
			Next    string          `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		correspondents = append(correspondents, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return correspondents, nil
}

func getDocumentTypes() ([]DocumentType, error) {
	var documentTypes []DocumentType
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/document_types/?page=%d", Config.Paperless.InstanceURL, page))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch document types: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []DocumentType `json:"results"`
			Next    string         `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		documentTypes = append(documentTypes, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return documentTypes, nil
}

func getStoragePaths() ([]StoragePath, error) {
	var storagePaths []StoragePath
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/storage_paths/?page=%d", Config.Paperless.InstanceURL, page))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch storage paths: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []StoragePath `json:"results"`
			Next    string        `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		storagePaths = append(storagePaths, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return storagePaths, nil
}

func getUsers() ([]User, error) {
	var users []User
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/users/?page=%d", Config.Paperless.InstanceURL, page))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch users: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []User `json:"results"`
			Next    string `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		users = append(users, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return users, nil
}

func getTags() ([]Tag, error) {
	var tags []Tag
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/tags/?page=%d", Config.Paperless.InstanceURL, page))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tags: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []Tag  `json:"results"`
			Next    string `json:"next"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		tags = append(tags, result.Results...)

		if result.Next == "" {
			break
		}
		page++
	}

	return tags, nil
}

func addMetaData(document *Document) error {
	resp, err := getRequest(fmt.Sprintf("%sapi/documents/%d/metadata/", Config.Paperless.InstanceURL, document.ID))
	if err != nil {
		return fmt.Errorf("failed to fetch meta data for document id=%d: %v", document.ID, err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(document)
	if err != nil {
		return err
	}

	return nil
}

func getDocumentsByTag(tag Tag, processedTag Tag) ([]Document, error) {
	var documents []Document
	page := 1

	for {
		resp, err := getRequest(fmt.Sprintf("%sapi/documents/?page=%d&tags__id__all=%d&tags__id__none=%d", Config.Paperless.InstanceURL, page, tag.ID, processedTag.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch documents: %v", err)
		}
		defer resp.Body.Close()

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

	// add meta data to each document
	for idx := range documents {
		if err := addMetaData(&documents[idx]); err != nil {
			fmt.Printf("error fetching meta data: %v", err)
			return nil, err
		}
	}

	return documents, nil
}

func addTagToDocument(document Document, tag Tag) error {
	url := fmt.Sprintf("%sapi/documents/bulk_edit/", Config.Paperless.InstanceURL)

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
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", Config.Paperless.InstanceToken))

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
	original := ""
	if Config.Paperless.DownloadOriginal {
		original = "?original=true"
	}
	url := fmt.Sprintf("%sapi/documents/%d/download/%s", Config.Paperless.InstanceURL, doc.ID, original)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", Config.Paperless.InstanceToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download document ID %d: %s", doc.ID, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getCorrespondentByID(correspondents []Correspondent, id int) *Correspondent {
	for _, correspondent := range correspondents {
		if correspondent.ID == id {
			return &correspondent
		}
	}
	return nil
}

func getDocumentTypeByID(documentTypes []DocumentType, id int) *DocumentType {
	for _, documentType := range documentTypes {
		if documentType.ID == id {
			return &documentType
		}
	}
	return nil
}

func getStoragePathByID(storagePaths []StoragePath, id int) *StoragePath {
	for _, storagePath := range storagePaths {
		if storagePath.ID == id {
			return &storagePath
		}
	}
	return nil
}

func getUserByID(users []User, id int) *User {
	for _, user := range users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

func getTagByID(tags []Tag, id int) *Tag {
	for _, tag := range tags {
		if tag.ID == id {
			return &tag
		}
	}
	return nil
}

func getTagByName(tags []Tag, name string) *Tag {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag
		}
	}
	return nil
}
