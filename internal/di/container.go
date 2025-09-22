package di

import (
	"log"
	"s3browser/internal/auth"
	"s3browser/internal/config"
	"s3browser/internal/handlers"
	"s3browser/internal/queue"
	"s3browser/internal/queue/queuedelete"
	"s3browser/internal/queue/queuepublish"
	"s3browser/internal/queue/queueunpublish"
	"s3browser/internal/queue/queueupload"
	"s3browser/internal/s3"
	"s3browser/internal/scheduler"
	"sync"
)

// Container holds all application dependencies
type Container struct {
	mu sync.RWMutex

	// Core dependencies
	config *config.Config

	// Services
	s3Service  *s3.S3Service
	jwtService *auth.JWTService

	// Shared database
	sharedQueue *queue.Queue

	// Queues
	uploadQueue    *queueupload.Queue
	publishQueue   *queuepublish.Queue
	unpublishQueue *queueunpublish.Queue
	deleteQueue    *queuedelete.Queue

	// Handlers and Scheduler
	handlers  *handlers.Handlers
	scheduler *scheduler.Scheduler

	// Initialization flags
	initialized bool
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	return &Container{}
}

// Initialize initializes all dependencies in the correct order
func (c *Container) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	// 1. Load configuration
	if err := c.initConfig(); err != nil {
		return err
	}

	// 2. Initialize S3 service
	if err := c.initS3Service(); err != nil {
		return err
	}

	// 3. Initialize JWT service
	if err := c.initJWTService(); err != nil {
		return err
	}

	// 4. Initialize shared database
	if err := c.initSharedDatabase(); err != nil {
		return err
	}

	// 5. Initialize queues
	if err := c.initQueues(); err != nil {
		return err
	}

	// 5. Initialize handlers
	if err := c.initHandlers(); err != nil {
		return err
	}

	// 6. Initialize scheduler
	if err := c.initScheduler(); err != nil {
		return err
	}

	c.initialized = true
	return nil
}

// initConfig loads the configuration
func (c *Container) initConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	c.config = cfg
	return nil
}

// initS3Service initializes the S3 service
func (c *Container) initS3Service() error {
	s3Service, err := s3.NewS3Service(
		c.config.S3.AccessKey,
		c.config.S3.SecretKey,
		c.config.S3.Region,
		c.config.S3.Endpoint,
		c.config.S3.Bucket,
	)
	if err != nil {
		return err
	}
	c.s3Service = s3Service
	return nil
}

// initJWTService initializes the JWT service
func (c *Container) initJWTService() error {
	users := make([]auth.User, 0, len(c.config.Users))
	for _, u := range c.config.Users {
		users = append(users, auth.User{
			Username: u.Username,
			Password: u.Password,
			Role:     u.Role,
		})
	}

	c.jwtService = auth.NewJWTService(c.config.JWT.Secret, users)
	return nil
}

// initSharedDatabase initializes the shared database
func (c *Container) initSharedDatabase() error {
	bucketNames := []string{
		"queue_upload",
		"queue_publish",
		"queue_unpublish",
		"queue_delete",
	}

	sharedQueue, err := queue.NewSharedQueueStore("data/queue.db", bucketNames)
	if err != nil {
		return err
	}

	c.sharedQueue = sharedQueue
	return nil
}

// initQueues initializes all queue services
func (c *Container) initQueues() error {
	// Upload queue
	c.uploadQueue = &queueupload.Queue{
		GenericQueue: queue.NewGenericQueueWithDB[*queue.UploadItem](c.sharedQueue, "queue_upload"),
	}

	// Publish queue
	c.publishQueue = &queuepublish.Queue{
		GenericQueue: queue.NewGenericQueueWithDB[*queue.PublishItem](c.sharedQueue, "queue_publish"),
	}

	// Unpublish queue
	c.unpublishQueue = &queueunpublish.Queue{
		GenericQueue: queue.NewGenericQueueWithDB[*queue.UnpublishItem](c.sharedQueue, "queue_unpublish"),
	}

	// Delete queue
	c.deleteQueue = &queuedelete.Queue{
		GenericQueue: queue.NewGenericQueueWithDB[*queue.DeleteItem](c.sharedQueue, "queue_delete"),
	}

	return nil
}

// initHandlers initializes the handlers
func (c *Container) initHandlers() error {
	c.handlers = handlers.NewHandlers(
		c.s3Service,
		c.uploadQueue,
		c.publishQueue,
		c.unpublishQueue,
		c.deleteQueue,
		c.jwtService,
	)
	return nil
}

// initScheduler initializes the scheduler
func (c *Container) initScheduler() error {
	sched, err := scheduler.NewScheduler(
		c.config,
		c.s3Service,
		c.uploadQueue,
		c.publishQueue,
		c.unpublishQueue,
		c.deleteQueue,
	)
	if err != nil {
		return err
	}

	c.scheduler = sched
	return nil
}

// Getters for dependencies (thread-safe)
func (c *Container) Config() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *Container) S3Service() *s3.S3Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.s3Service
}

func (c *Container) JWTService() *auth.JWTService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtService
}

func (c *Container) UploadQueue() *queueupload.Queue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uploadQueue
}

func (c *Container) PublishQueue() *queuepublish.Queue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.publishQueue
}

func (c *Container) UnpublishQueue() *queueunpublish.Queue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.unpublishQueue
}

func (c *Container) DeleteQueue() *queuedelete.Queue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deleteQueue
}

func (c *Container) Handlers() *handlers.Handlers {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.handlers
}

func (c *Container) Scheduler() *scheduler.Scheduler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.scheduler
}

// StartScheduler starts the scheduler
func (c *Container) StartScheduler() error {
	c.mu.RLock()
	sched := c.scheduler
	c.mu.RUnlock()

	if sched == nil {
		log.Fatal("Scheduler not initialized")
	}

	return sched.Start()
}

// StopScheduler stops the scheduler
func (c *Container) StopScheduler() error {
	c.mu.RLock()
	sched := c.scheduler
	c.mu.RUnlock()

	if sched == nil {
		return nil
	}

	return sched.Stop()
}

// CloseQueues closes all queues
func (c *Container) CloseQueues() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Close the shared database (this closes all queues)
	if c.sharedQueue != nil {
		c.sharedQueue.Close()
	}
}

// IsInitialized returns whether the container has been initialized
func (c *Container) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}
