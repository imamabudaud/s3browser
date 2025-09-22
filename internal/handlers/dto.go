package handlers

import (
	"fmt"
	"strings"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type FileItem struct {
	Name         string `json:"name"`
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
	IsDir        bool   `json:"isDir"`
	IsPublic     bool   `json:"isPublic"`
	PublicURL    string `json:"publicUrl,omitempty"`
}

type UploadRequest struct {
	Files []string `json:"files"` // For CSV upload, this will be the CSV content
	Type  string   `json:"type"`  // "file" or "csv"
}

type UploadResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

type BatchUploadItem struct {
	ID              string `json:"id"`
	RemoteURL       string `json:"remoteUrl"`
	DestinationName string `json:"destinationName"`
	TargetFolder    string `json:"targetFolder"`
	Status          string `json:"status"`
}

type BatchPublicItem struct {
	ID       string `json:"id"`
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
	Status   string `json:"status"`
}

type batchActionRequest struct {
	Keys []string `json:"keys"`
}

func (d batchActionRequest) validate() error {
	if len(d.Keys) == 0 {
		return fmt.Errorf("No keys provided for delete objects")
	}

	for _, key := range d.Keys {
		if key == "" {
			return fmt.Errorf("Empty key provided")
		}
	}

	return nil
}

type keyedObject struct {
	FilePath string
	FileName string
}

func (d batchActionRequest) parseKeys() ([]keyedObject, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}

	var parsedKeys []keyedObject
	for _, key := range d.Keys {
		fileName := key
		if lastSlash := strings.LastIndex(key, "/"); lastSlash != -1 {
			fileName = key[lastSlash+1:]
		}

		parsedKeys = append(parsedKeys, keyedObject{
			FilePath: key,
			FileName: fileName,
		})
	}

	return parsedKeys, nil
}
