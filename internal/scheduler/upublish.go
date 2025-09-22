package scheduler

import (
	"fmt"
	"log"
	"s3browser/internal/config"
	"s3browser/internal/queue"
	"s3browser/internal/queue/queueunpublish"
	"s3browser/internal/s3"
	"sync"
)

type unpublishProcessor struct {
	processingMux sync.Mutex
	isProcessing  bool
	queue         *queueunpublish.Queue
	config        *config.Config
	s3Service     *s3.S3Service
}

func newUnpublishProcessor(cfg *config.Config, service *s3.S3Service, q *queueunpublish.Queue) *unpublishProcessor {
	return &unpublishProcessor{
		processingMux: sync.Mutex{},
		isProcessing:  false,
		queue:         q,
		config:        cfg,
		s3Service:     service,
	}
}

func (p *unpublishProcessor) process() {
	// Try to acquire the processing lock
	p.processingMux.Lock()
	if p.isProcessing {
		p.processingMux.Unlock()
		log.Println("Unpublish processing already in progress, skipping this run")
		return
	}
	p.isProcessing = true
	p.processingMux.Unlock()

	// Ensure we always release the lock when done
	defer func() {
		p.processingMux.Lock()
		p.isProcessing = false
		p.processingMux.Unlock()
		log.Println("Unpublish processing batch completed")
	}()

	log.Println("Processing unpublish queue items...")

	// Pop all available items at once (up to MaxConcurrentUnpublish)
	items := p.queue.PopN(p.config.Queue.MaxConcurrentUnpublish)
	if len(items) == 0 {
		return
	}

	log.Printf("Found %d pending unpublish items to process", len(items))

	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(unpublishItem *queueunpublish.Item) {
			defer wg.Done()
			p.processItem(unpublishItem)
		}(&items[i])
	}

	wg.Wait()
}

func (p *unpublishProcessor) processItem(item *queueunpublish.Item) {
	log.Printf("Unpublishing: %s", item.FilePath)

	err := p.s3Service.MakePrivate(item.FilePath)
	if err != nil {
		fmt.Printf("Failed to make private: %v", err)
		if err = p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	if err := p.queue.UpdateStatus(item.ID, queue.StatusSuccess); err != nil {
		fmt.Printf("Error update status: %v", err)
		return
	}

	log.Printf("Successfully unpublishing: %s", item.FileName)
}
