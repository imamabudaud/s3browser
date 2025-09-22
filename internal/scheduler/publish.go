package scheduler

import (
	"fmt"
	"log"
	"s3browser/internal/config"
	"s3browser/internal/queue"
	"s3browser/internal/queue/queuepublish"
	"s3browser/internal/s3"
	"sync"
)

type publishProcessor struct {
	processingMux sync.Mutex
	isProcessing  bool
	queue         *queuepublish.Queue
	config        *config.Config
	s3Service     *s3.S3Service
}

func newPublishProcessor(cfg *config.Config, service *s3.S3Service, q *queuepublish.Queue) *publishProcessor {
	return &publishProcessor{
		processingMux: sync.Mutex{},
		isProcessing:  false,
		queue:         q,
		config:        cfg,
		s3Service:     service,
	}
}

func (p *publishProcessor) process() {
	// Try to acquire the processing lock
	p.processingMux.Lock()
	if p.isProcessing {
		p.processingMux.Unlock()
		log.Println("Publish processing already in progress, skipping this run")
		return
	}
	p.isProcessing = true
	p.processingMux.Unlock()

	// Ensure we always release the lock when done
	defer func() {
		p.processingMux.Lock()
		p.isProcessing = false
		p.processingMux.Unlock()
		log.Println("Publish processing batch completed")
	}()

	log.Println("Processing publish queue items...")

	// Pop all available items at once (up to MaxConcurrentPublish)
	items := p.queue.PopN(p.config.Queue.MaxConcurrentPublish)
	if len(items) == 0 {
		return
	}

	log.Printf("Found %d pending publish items to process", len(items))

	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(publishItem *queuepublish.Item) {
			defer wg.Done()
			p.processItem(publishItem)
		}(&items[i])
	}

	wg.Wait()
}

func (p *publishProcessor) processItem(item *queuepublish.Item) {
	log.Printf("Publishing: %s", item.FilePath)

	err := p.s3Service.MakePublic(item.FilePath)
	if err != nil {
		fmt.Printf("Failed to make public: %v", err)
		if err = p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	if err := p.queue.UpdateStatus(item.ID, queue.StatusSuccess); err != nil {
		fmt.Printf("Error update status: %v", err)
		return
	}

	log.Printf("Successfully publishing: %s", item.FileName)
}
