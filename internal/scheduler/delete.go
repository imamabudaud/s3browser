package scheduler

import (
	"fmt"
	"log"
	"s3browser/internal/config"
	"s3browser/internal/queue"
	"s3browser/internal/queue/queuedelete"
	"s3browser/internal/s3"
	"sync"
)

type deleteProcessor struct {
	processingMux sync.Mutex
	isProcessing  bool
	queue         *queuedelete.Queue
	config        *config.Config
	s3Service     *s3.S3Service
}

func newDeleteProcessor(cfg *config.Config, service *s3.S3Service, q *queuedelete.Queue) *deleteProcessor {
	return &deleteProcessor{
		processingMux: sync.Mutex{},
		isProcessing:  false,
		queue:         q,
		config:        cfg,
		s3Service:     service,
	}
}

func (p *deleteProcessor) process() {
	// Try to acquire the processing lock
	p.processingMux.Lock()
	if p.isProcessing {
		p.processingMux.Unlock()
		log.Println("Delete processing already in progress, skipping this run")
		return
	}
	p.isProcessing = true
	p.processingMux.Unlock()

	// Ensure we always release the lock when done
	defer func() {
		p.processingMux.Lock()
		p.isProcessing = false
		p.processingMux.Unlock()
		log.Println("Delete processing batch completed")
	}()

	log.Println("Processing delete queue items...")

	// Pop all available items at once (up to MaxConcurrentDelete)
	items := p.queue.PopN(p.config.Queue.MaxConcurrentDelete)
	if len(items) == 0 {
		return
	}

	log.Printf("Found %d pending delete items to process", len(items))

	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(deleteItem *queuedelete.Item) {
			defer wg.Done()
			p.processItem(deleteItem)
		}(&items[i])
	}

	wg.Wait()
}

func (p *deleteProcessor) processItem(item *queuedelete.Item) {
	log.Printf("Publishing: %s", item.FilePath)

	err := p.s3Service.DeleteObject(item.FilePath)
	if err != nil {
		fmt.Printf("Failed to delete object: %v", err)
		if err = p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	if err := p.queue.UpdateStatus(item.ID, queue.StatusSuccess); err != nil {
		fmt.Printf("Error update status: %v", err)
		return
	}

	log.Printf("Successfully delete: %s", item.FileName)
}
