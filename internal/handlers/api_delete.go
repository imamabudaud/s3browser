package handlers

import (
	"net/http"
	"s3browser/internal/queue"

	"github.com/labstack/echo/v4"
)

func (h *Handlers) deleteObjectSync(item keyedObject) error {
	err := h.s3Service.DeleteObject(item.FilePath)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) DeleteObjects(c echo.Context) error {
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
		err := h.deleteObjectSync(parsedKeys[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   err.Error(),
			})
		}

		return c.JSON(http.StatusOK, Response{
			Success: true,
			Data: UploadResponse{
				Message: "Files deleted successfully",
			},
		})
	}

	addedCount := 0
	for _, key := range parsedKeys {
		item := queue.DeleteItem{
			FilePath: key.FilePath,
			FileName: key.FileName,
		}
		if err := h.deleteQueue.Push(item); err == nil {
			addedCount++
		}
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: UploadResponse{
			Message: "Files added to delete queue successfully",
			Count:   addedCount,
		},
	})
}

func (h *Handlers) ListPendingDeletes(c echo.Context) error {
	pendingPublishes := h.deleteQueue.All()

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

func (h *Handlers) ClearQueueDeletes(c echo.Context) error {
	if err := h.deleteQueue.Clear(); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to clear queue delete items: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "All delete queue items cleared successfully"},
	})
}
