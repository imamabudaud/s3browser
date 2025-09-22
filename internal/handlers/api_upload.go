package handlers

import (
	"encoding/csv"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"s3browser/internal/queue/queueupload"
	"strings"
)

func (h *Handlers) UploadObject(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Error parsing form: " + err.Error(),
		})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "No files selected",
		})
	}

	prefix := c.FormValue("prefix")
	uploadedCount := 0

	for _, file := range files {
		key := prefix + file.Filename
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			key = prefix + "/" + file.Filename
		}

		src, err := file.Open()
		if err != nil {
			continue
		}

		err = h.s3Service.PutObject(key, src)
		src.Close()

		if err != nil {
			continue
		}
		uploadedCount++
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: UploadResponse{
			Message: "Files uploaded successfully",
			Count:   uploadedCount,
		},
	})
}

func (h *Handlers) RemoteUploads(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Error parsing form: " + err.Error(),
		})
	}

	files := form.File["csvFile"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "No CSV file selected",
		})
	}

	prefix := c.FormValue("prefix")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	file := files[0]
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Error opening CSV file: " + err.Error(),
		})
	}
	defer src.Close()

	// Parse CSV and add to queue (reuse existing logic)
	records, err := parseCSV(src)
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Error parsing CSV: " + err.Error(),
		})
	}

	addedCount := 0
	for _, record := range records {
		item := queueupload.Item{
			RemoteURL:       record.RemoteURL,
			DestinationName: record.DestinationName,
			TargetFolder:    prefix,
		}
		if err := h.queueUpload.Push(item); err == nil {
			addedCount++
		}
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: UploadResponse{
			Message: "CSV processed successfully",
			Count:   addedCount,
		},
	})
}

func (h *Handlers) ListPendingUploads(c echo.Context) error {
	uploads := h.queueUpload.All()

	apiUploads := []BatchUploadItem{}
	for _, upload := range uploads {
		apiUploads = append(apiUploads, BatchUploadItem{
			ID:              upload.ID,
			RemoteURL:       upload.RemoteURL,
			DestinationName: upload.DestinationName,
			TargetFolder:    upload.TargetFolder,
			Status:          string(upload.Status),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    apiUploads,
	})
}

func (h *Handlers) ClearQueueUploads(c echo.Context) error {
	if err := h.queueUpload.Clear(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to clear queue uploads: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "All queue uploads cleared successfully"},
	})
}

// Helper function to parse CSV (reuse from existing handler)
func parseCSV(src io.Reader) ([]struct {
	RemoteURL       string
	DestinationName string
}, error) {
	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, errors.New("CSV must have at least a header row and one data row")
	}

	var results []struct {
		RemoteURL       string
		DestinationName string
	}

	// Skip header row and process data rows
	for _, record := range records[1:] {
		if len(record) < 1 {
			continue // Skip empty rows
		}

		remoteURL := strings.TrimSpace(record[0])
		if remoteURL == "" {
			continue // Skip rows without remote URL
		}

		// Validate URL
		if _, err := url.Parse(remoteURL); err != nil {
			continue // Skip invalid URLs
		}

		destinationName := ""
		if len(record) > 1 {
			destinationName = strings.TrimSpace(record[1])
		}

		// If destination name is empty, extract from URL
		if destinationName == "" {
			parsedURL, _ := url.Parse(remoteURL)
			destinationName = filepath.Base(parsedURL.Path)
		}

		results = append(results, struct {
			RemoteURL       string
			DestinationName string
		}{
			RemoteURL:       remoteURL,
			DestinationName: destinationName,
		})
	}

	return results, nil
}
