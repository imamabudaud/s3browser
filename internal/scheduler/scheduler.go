package scheduler

import (
	"log"
	"s3browser/internal/config"
	"s3browser/internal/queue/queuedelete"
	"s3browser/internal/queue/queuepublish"
	"s3browser/internal/queue/queueunpublish"
	"s3browser/internal/queue/queueupload"
	"s3browser/internal/s3"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Scheduler struct {
	uploadQueue    *queueupload.Queue
	publishQueue   *queuepublish.Queue
	unpublishQueue *queueunpublish.Queue
	deleteQueue    *queuedelete.Queue
	scheduler      gocron.Scheduler
	s3Service      *s3.S3Service
	config         *config.Config
	processingMux  sync.Mutex
	isProcessing   bool
}

func NewScheduler(cfg *config.Config, s3Service *s3.S3Service, uploadQueue *queueupload.Queue, publishQueue *queuepublish.Queue, unpublishQueue *queueunpublish.Queue, deleteQueue *queuedelete.Queue) (*Scheduler, error) {

	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		uploadQueue:    uploadQueue,
		publishQueue:   publishQueue,
		unpublishQueue: unpublishQueue,
		deleteQueue:    deleteQueue,
		scheduler:      s,
		s3Service:      s3Service,
		config:         cfg,
	}, nil
}

func (s *Scheduler) Start() error {
	// Add upload processing job
	uploadInterval, err := s.config.GetUploadInterval()
	if err != nil {
		return err
	}

	uploadProc := newUploadProcessor(s.config, s.s3Service, s.uploadQueue)
	_, err = s.scheduler.NewJob(
		gocron.DurationJob(uploadInterval),
		gocron.NewTask(uploadProc.process),
		gocron.WithName("upload-processor"),
	)
	if err != nil {
		return err
	}

	publishProc := newPublishProcessor(s.config, s.s3Service, s.publishQueue)
	_, err = s.scheduler.NewJob(
		gocron.DurationJob(5*time.Second), // Process public items every 5 seconds
		gocron.NewTask(publishProc.process),
		gocron.WithName("publish-processor"),
	)
	if err != nil {
		return err
	}

	unpublishProc := newUnpublishProcessor(s.config, s.s3Service, s.unpublishQueue)
	_, err = s.scheduler.NewJob(
		gocron.DurationJob(5*time.Second), // Process public items every 5 seconds
		gocron.NewTask(unpublishProc.process),
		gocron.WithName("unpublish-processor"),
	)
	if err != nil {
		return err
	}

	deleteProc := newDeleteProcessor(s.config, s.s3Service, s.deleteQueue)
	_, err = s.scheduler.NewJob(
		gocron.DurationJob(5*time.Second), // Process public items every 5 seconds
		gocron.NewTask(deleteProc.process),
		gocron.WithName("delete-processor"),
	)
	if err != nil {
		return err
	}

	// Start the scheduler
	s.scheduler.Start()

	log.Printf("Scheduler started with upload interval: %v", uploadInterval)
	return nil
}

func (s *Scheduler) Stop() error {
	return s.scheduler.Shutdown()
}

// IsProcessing returns true if upload processing is currently active
func (s *Scheduler) IsProcessing() bool {
	s.processingMux.Lock()
	defer s.processingMux.Unlock()
	return s.isProcessing
}
