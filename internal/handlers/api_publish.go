package handlers

import (
	"fmt"
	"net/http"
	"s3browser/internal/queue/queuepublish"
	"strings"

	"github.com/labstack/echo/v4"
)

func (h *Handlers) publishObjectSync(item keyedObject) error {
	err := h.s3Service.MakePublic(item.FilePath)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) PublishObjects(c echo.Context) error {
	var req batchActionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	parsedKeys, err := req.parseKeys()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
	}

	if len(req.Keys) == 1 {
		err := h.publishObjectSync(parsedKeys[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   err.Error(),
			})
		}

		return c.JSON(http.StatusOK, Response{
			Success: true,
			Data: UploadResponse{
				Message: "Files published successfully",
			},
		})
	}

	// Add each file to the public queue
	addedCount := 0
	for _, key := range parsedKeys {
		item := queuepublish.Item{
			FilePath: key.FilePath,
			FileName: key.FileName,
		}
		if err := h.publishQueue.Push(item); err == nil {
			addedCount++
		}
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: UploadResponse{
			Message: "Files added to public queue successfully",
			Count:   addedCount,
		},
	})
}

func (h *Handlers) ListPendingPublishes(c echo.Context) error {
	pendingPublishes := h.publishQueue.All()

	// Convert to API format
	apiPublics := []BatchPublicItem{}
	for _, public := range pendingPublishes {
		apiPublics = append(apiPublics, BatchPublicItem{
			ID:       public.ID,
			FilePath: public.FilePath,
			FileName: public.FileName,
			Status:   string(public.Status),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    apiPublics,
	})
}

func (h *Handlers) ExportPublishedObjects(c echo.Context) error {
	var req batchActionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	parsedKeys, err := req.parseKeys()
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
	}

	// Create CSV content
	var csvContent strings.Builder
	csvContent.WriteString("file_path,file_name,public_url\n")

	for _, keyObj := range parsedKeys {
		// Check if the object is public and get its public URL
		isPublic, publicURL := h.s3Service.IsObjectPublic(keyObj.FilePath)

		// Escape CSV fields
		filePath := strings.ReplaceAll(keyObj.FilePath, `"`, `""`)
		fileName := strings.ReplaceAll(keyObj.FileName, `"`, `""`)

		if isPublic {
			publicURL = strings.ReplaceAll(publicURL, `"`, `""`)
		} else {
			publicURL = "" // Empty if not public
		}

		csvContent.WriteString(fmt.Sprintf(`"%s","%s","%s"`+"\n", filePath, fileName, publicURL))
	}

	// Set response headers for CSV download
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=selected_files_public_urls.csv")

	return c.String(http.StatusOK, csvContent.String())
}

func (h *Handlers) ClearQueuePublishes(c echo.Context) error {
	if err := h.publishQueue.Clear(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to clear queue publish items: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "All publish queue items cleared successfully"},
	})
}
