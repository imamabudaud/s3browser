package scheduler

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"s3browser/internal/config"
	"s3browser/internal/queue"
	"s3browser/internal/queue/queueupload"
	"s3browser/internal/s3"
	"strings"
	"sync"
	"time"
)

type uploadProcessor struct {
	processingMux sync.Mutex
	isProcessing  bool
	queue         *queueupload.Queue
	config        *config.Config
	s3Service     *s3.S3Service
}

func newUploadProcessor(cfg *config.Config, service *s3.S3Service, q *queueupload.Queue) *uploadProcessor {
	return &uploadProcessor{
		processingMux: sync.Mutex{},
		isProcessing:  false,
		queue:         q,
		config:        cfg,
		s3Service:     service,
	}
}

func (p *uploadProcessor) process() {
	// Try to acquire the processing lock
	p.processingMux.Lock()
	if p.isProcessing {
		p.processingMux.Unlock()
		log.Println("Upload processing already in progress, skipping this run")
		return
	}
	p.isProcessing = true
	p.processingMux.Unlock()

	// Ensure we always release the lock when done
	defer func() {
		p.processingMux.Lock()
		p.isProcessing = false
		p.processingMux.Unlock()
		log.Println("Upload processing batch completed")
	}()

	log.Println("Processing uploads...")

	// Pop all available items at once (up to MaxConcurrentUpload)
	items := p.queue.PopN(p.config.Queue.MaxConcurrentUpload)
	if len(items) == 0 {
		log.Println("No pending uploads found")
		return
	}

	log.Printf("Found %d pending uploads to process", len(items))

	// Use a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Process all collected uploads
	for i := range items {
		wg.Add(1)
		go func(uploadItem *queueupload.Item) {
			defer wg.Done()
			p.processItem(uploadItem)
		}(&items[i])
	}

	// Wait for all uploads in this batch to complete
	wg.Wait()
}

// processItem processes a single upload item
func (p *uploadProcessor) processItem(item *queueupload.Item) {
	log.Printf("Processing upload: %s -> %s", item.RemoteURL, item.DestinationName)

	client := p.createHTTPClient()

	// Download file from remote URL
	resp, err := client.Get(item.RemoteURL)
	if err != nil {
		if err := p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update upload status: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if err := p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	// Read the response body into memory to make it seekable for S3 retries
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		if err := p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	// Create S3 key
	var key string
	if item.TargetFolder == "" {
		key = item.DestinationName
	} else if strings.HasSuffix(item.TargetFolder, "/") {
		key = item.TargetFolder + item.DestinationName
	} else {
		key = item.TargetFolder + "/" + item.DestinationName
	}

	err = p.s3Service.PutObject(key, bytes.NewReader(bodyData))
	if err != nil {
		if err := p.queue.UpdateStatus(item.ID, queue.StatusFailed); err != nil {
			fmt.Printf("Error update status: %v", err)
		}
		return
	}

	if err := p.queue.UpdateStatus(item.ID, queue.StatusSuccess); err != nil {
		log.Printf("Error update status upload: %v", err)
	} else {
		log.Printf("Successfully processed upload: %s", item.DestinationName)
	}
}

func (p *uploadProcessor) createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if p.config.Queue.SkipTLSVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client.Transport = transport
	}

	return client
}
