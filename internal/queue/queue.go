package queue

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusSuccess    Status = "SUCCESS"
	StatusFailed     Status = "FAILED"
)

var (
	ErrBucketNotFound = fmt.Errorf("bucket not found")
	ErrItemNotFound   = fmt.Errorf("item not found")
)

type Queue struct {
	db   *bbolt.DB
	path string
}

func (q *Queue) Update(fn func(tx *bbolt.Tx) error) error {
	return q.db.Update(fn)
}
func (q *Queue) Read(fn func(tx *bbolt.Tx) error) error {
	return q.db.View(fn)
}

func NewQueueStore(dbPath, bucketName string) (*Queue, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open the database with retry logic
	var db *bbolt.DB
	var err error

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{
			Timeout:    30 * time.Second,
			NoGrowSync: false,
			ReadOnly:   false,
		})
		if err == nil {
			break
		}

		log.Printf("Attempt %d to open database failed: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
	}

	// Create the buckets if they don't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		_, createErr := tx.CreateBucketIfNotExists([]byte(bucketName))
		if createErr != nil {
			return createErr
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Queue{
		db:   db,
		path: dbPath,
	}, nil
}

func (q *Queue) Close() error {
	return q.db.Close()
}

func GenerateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	randomString := string(b)

	return time.Now().Format("20060102150405") + "-" + randomString
}

// NewSharedQueueStore creates a shared database instance with multiple buckets
func NewSharedQueueStore(dbPath string, bucketNames []string) (*Queue, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open the database with retry logic
	var db *bbolt.DB
	var err error

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{
			Timeout:    5 * time.Second,
			NoGrowSync: false,
			ReadOnly:   false,
		})
		if err == nil {
			break
		}

		log.Printf("Attempt %d to open database failed: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
	}

	// Create all the buckets if they don't exist
	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucketName := range bucketNames {
			_, createErr := tx.CreateBucketIfNotExists([]byte(bucketName))
			if createErr != nil {
				return createErr
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &Queue{
		db:   db,
		path: dbPath,
	}, nil
}
