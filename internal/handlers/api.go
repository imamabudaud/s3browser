package handlers

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handlers) ServeFile(c echo.Context) error {
	key := c.Param("*")
	key = strings.TrimPrefix(key, "/")

	// URL decode the key to handle encoded characters like %2F
	decodedKey, decodeErr := url.QueryUnescape(key)
	if decodeErr != nil {
		return c.String(http.StatusBadRequest, "Invalid file path")
	}
	key = decodedKey

	token := c.QueryParam("token")
	if token == "" {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Authorization required",
			})
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid authorization format",
			})
		}
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	_, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid or expired token",
		})
	}

	obj, err := h.s3Service.GetObject(key)
	if err != nil {
		return c.String(http.StatusNotFound, "File not found")
	}
	defer obj.Body.Close()

	// Set the appropriate content type
	contentType := "application/octet-stream"
	if obj.ContentType != nil {
		contentType = *obj.ContentType
	}

	// Detect content type from file extension if not set
	if contentType == "application/octet-stream" || contentType == "binary/octet-stream" {
		detectedType := mime.TypeByExtension(path.Ext(key))
		if detectedType != "" {
			contentType = detectedType
		}
	}

	// Set content disposition to inline so files open in browser instead of downloading
	filename := path.Base(key)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Response().Header().Set("Content-Type", contentType)

	// Stream the file directly to the response
	_, err = io.Copy(c.Response(), obj.Body)
	return err
}

func (h *Handlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	token, err := h.jwtService.GenerateToken(req.Username, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	user, _ := h.jwtService.GetUserFromToken(token)

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: LoginResponse{
			Token: token,
			User: User{
				Username: user.Username,
				Role:     user.Role,
			},
		},
	})
}

func (h *Handlers) ListObjects(c echo.Context) error {
	prefix := c.QueryParam("prefix")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	files, err := h.s3Service.ListObjects(prefix)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to list files: " + err.Error(),
		})
	}

	// Convert to API format
	var apiFiles []FileItem
	for _, file := range files {
		apiFiles = append(apiFiles, FileItem{
			Name:         file.Name,
			Key:          file.Key,
			Size:         file.Size,
			LastModified: file.LastModified,
			IsDir:        file.IsDir,
			IsPublic:     file.IsPublic,
			PublicURL:    file.PublicURL,
		})
	}

	// Sort: folders first (alphabetically), then files (alphabetically)
	sort.Slice(apiFiles, func(i, j int) bool {
		// If one is a directory and the other isn't, the directory comes first
		if apiFiles[i].IsDir != apiFiles[j].IsDir {
			return apiFiles[i].IsDir
		}

		// If both are the same type (both dirs or both files), sort alphabetically by name
		return strings.ToLower(apiFiles[i].Name) < strings.ToLower(apiFiles[j].Name)
	})

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    apiFiles,
	})
}

func (h *Handlers) CreateFolder(c echo.Context) error {
	var req struct {
		FolderName string `json:"folderName"`
		Prefix     string `json:"prefix"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	if req.FolderName == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Folder name is required",
		})
	}

	key := req.Prefix + req.FolderName + "/"
	if req.Prefix != "" && !strings.HasSuffix(req.Prefix, "/") {
		key = req.Prefix + "/" + req.FolderName + "/"
	}

	err := h.s3Service.PutObject(key, strings.NewReader(""))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Error creating folder: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "Folder created successfully"},
	})
}

func (h *Handlers) GetFolderStats(c echo.Context) error {
	prefix := c.QueryParam("prefix")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	stats, err := h.s3Service.GetFolderStats(prefix)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to get folder statistics: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

func (h *Handlers) ImagePreview(c echo.Context) error {
	fileKey := c.Param("*")
	if fileKey == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "File key is required",
		})
	}

	// URL decode the file key to handle encoded characters like %2F
	decodedKey, decodeErr := url.QueryUnescape(fileKey)
	if decodeErr != nil {
		fmt.Printf("DEBUG: Error decoding URL: %v\n", decodeErr)
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid file path encoding",
		})
	}
	fileKey = decodedKey

	// Debug logging (can be removed in production)
	// fmt.Printf("DEBUG: Requested file key (decoded): %s\n", fileKey)

	// Check if the file is an image
	if !h.s3Service.IsImageFile(fileKey) {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "File is not an image",
		})
	}

	// Check if file exists and get its ACL directly
	obj, err := h.s3Service.GetObject(fileKey)
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "File not found",
		})
	}
	defer obj.Body.Close()

	// Check if file is public
	isPublic, publicURL := h.s3Service.IsObjectPublic(fileKey)

	var imageURL string
	if isPublic && publicURL != "" {
		imageURL = publicURL
	} else {
		// Generate signed URL with 24 hour expiration
		signedURL, err := h.s3Service.GenerateSignedURL(fileKey, 24*time.Hour)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   "Failed to generate signed URL: " + err.Error(),
			})
		}
		imageURL = signedURL
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"url": imageURL,
		},
	})
}
