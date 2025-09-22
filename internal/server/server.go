package server

import (
	"context"
	"net/http"
	"s3browser/internal/di"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server wraps the Echo server and DI container
type Server struct {
	container *di.Container
	echo      *echo.Echo
}

// NewServer creates a new server instance
func NewServer(container *di.Container) *Server {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return &Server{
		container: container,
		echo:      e,
	}
}

// SetupRoutes configures all routes
func (s *Server) SetupRoutes() {
	handlers := s.container.Handlers()
	jwtService := s.container.JWTService()

	// Web UI routes
	s.echo.GET("/login", func(c echo.Context) error {
		return c.File("ui/pages/login.html")
	})
	s.echo.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/browse")
	})
	s.echo.GET("/browse", func(c echo.Context) error {
		return c.File("ui/pages/files.html")
	})
	s.echo.GET("/pending-uploads", func(c echo.Context) error {
		return c.File("ui/pages/pending-uploads.html")
	})
	s.echo.GET("/pending-publics", func(c echo.Context) error {
		return c.File("ui/pages/batch-publics.html")
	})
	s.echo.GET("/pending-privates", func(c echo.Context) error {
		return c.File("ui/pages/batch-privates.html")
	})
	s.echo.GET("/pending-deletes", func(c echo.Context) error {
		return c.File("ui/pages/batch-deletes.html")
	})
	s.echo.GET("/preview", func(c echo.Context) error {
		return c.File("ui/pages/preview.html")
	})
	s.echo.GET("/files/*", func(c echo.Context) error {
		return handlers.ServeFile(c)
	})

	// Static assets for web UI
	s.echo.GET("/assets/*", func(c echo.Context) error {
		return c.File("ui/assets/" + c.Param("*"))
	})

	// API routes
	apiGroup := s.echo.Group("/api")

	// Authentication
	apiGroup.POST("/login", handlers.Login)
	apiGroup.GET("/list-objects", handlers.ListObjects, jwtService.JWTMiddleware())

	// Upload operations
	apiGroup.POST("/upload-object", handlers.UploadObject, jwtService.JWTMiddleware())
	apiGroup.POST("/remote-uploads", handlers.RemoteUploads, jwtService.JWTMiddleware())
	apiGroup.GET("/list-queue-uploads", handlers.ListPendingUploads, jwtService.JWTMiddleware())

	// Admin operations
	apiGroup.POST("/create-folder", handlers.CreateFolder, jwtService.JWTMiddleware(), jwtService.RequireAdmin())
	apiGroup.DELETE("/delete-objects", handlers.DeleteObjects, jwtService.JWTMiddleware(), jwtService.RequireAdmin())
	apiGroup.GET("/list-queue-deletes", handlers.ListPendingDeletes, jwtService.JWTMiddleware())

	// Publish operations
	apiGroup.POST("/publish-objects", handlers.PublishObjects, jwtService.JWTMiddleware())
	apiGroup.GET("/list-queue-publish", handlers.ListPendingPublishes, jwtService.JWTMiddleware())
	apiGroup.POST("/export-published-objects", handlers.ExportPublishedObjects, jwtService.JWTMiddleware())

	// Unpublish operations
	apiGroup.POST("/unpublish-objects", handlers.UnpublishObjects, jwtService.JWTMiddleware())
	apiGroup.GET("/list-queue-unpublish", handlers.ListPendingUnpublishes, jwtService.JWTMiddleware())

	// Queue management
	apiGroup.DELETE("/clear-queue-uploads", handlers.ClearQueueUploads, jwtService.JWTMiddleware())
	apiGroup.DELETE("/clear-queue-publish", handlers.ClearQueuePublishes, jwtService.JWTMiddleware())
	apiGroup.DELETE("/clear-queue-unpublish", handlers.ClearQueueUnpublishes, jwtService.JWTMiddleware())
	apiGroup.DELETE("/clear-queue-delete", handlers.ClearQueueDeletes, jwtService.JWTMiddleware())

	// Statistics and utilities
	apiGroup.GET("/get-folder-statistics", handlers.GetFolderStats, jwtService.JWTMiddleware())
	apiGroup.GET("/get-signed-url/*", handlers.ImagePreview, jwtService.JWTMiddleware())
}

// Start starts the server
func (s *Server) Start(addr string) error {
	return s.echo.Start(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.echo.Shutdown(ctx)
}

// Echo returns the underlying Echo instance
func (s *Server) Echo() *echo.Echo {
	return s.echo
}
