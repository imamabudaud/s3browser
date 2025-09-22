package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"s3browser/internal/queue"
)

func (h *Handlers) unpublishObjectSync(item keyedObject) error {
	err := h.s3Service.MakePrivate(item.FilePath)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) UnpublishObjects(c echo.Context) error {
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
		err := h.unpublishObjectSync(parsedKeys[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   err.Error(),
			})
		}

		return c.JSON(http.StatusOK, Response{
			Success: true,
			Data: UploadResponse{
				Message: "Files unpublished successfully",
			},
		})
	}

	// Add each file to the private queue
	addedCount := 0
	for _, key := range parsedKeys {
		item := queue.UnpublishItem{
			FilePath: key.FilePath,
			FileName: key.FileName,
		}
		if err := h.unpublishQueue.Push(item); err == nil {
			addedCount++
		}
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: UploadResponse{
			Message: "Files added to unpubilsh queue successfully",
			Count:   addedCount,
		},
	})
}

func (h *Handlers) ListPendingUnpublishes(c echo.Context) error {
	unpublishes := h.unpublishQueue.All()

	// Convert to API format
	apiPublics := []queue.UnpublishItem{}
	for _, public := range unpublishes {
		apiPublics = append(apiPublics, queue.UnpublishItem{
			BaseItem: queue.BaseItem{
				ID:     public.ID,
				Status: public.Status,
			},
			FilePath: public.FilePath,
			FileName: public.FileName,
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    apiPublics,
	})
}

func (h *Handlers) ClearQueueUnpublishes(c echo.Context) error {
	if err := h.unpublishQueue.Clear(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to clear queue unpublish items: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "All unpublish queue items cleared successfully"},
	})
}
