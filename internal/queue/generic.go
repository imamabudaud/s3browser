package queue

import (
	"encoding/json"
	"fmt"
	"log"

	"go.etcd.io/bbolt"
)

type Item interface {
	GetID() string
	SetID(string)
	GetStatus() Status
	SetStatus(Status)
}

type GenericQueue[T Item] struct {
	queue      *Queue
	bucketName string
}

func NewGenericQueue[T Item](dbPath, bucketName string) (*GenericQueue[T], error) {
	q, err := NewQueueStore(dbPath, bucketName)
	if err != nil {
		return nil, err
	}

	return &GenericQueue[T]{
		queue:      q,
		bucketName: bucketName,
	}, nil
}

// NewGenericQueueWithDB creates a new generic queue using an existing database
func NewGenericQueueWithDB[T Item](queue *Queue, bucketName string) *GenericQueue[T] {
	return &GenericQueue[T]{
		queue:      queue,
		bucketName: bucketName,
	}
}

func (gq *GenericQueue[T]) Close() error {
	return gq.queue.Close()
}

func (gq *GenericQueue[T]) Push(item T) error {
	item.SetID(GenerateID())
	item.SetStatus(StatusPending)

	return gq.queue.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}

		return bucket.Put([]byte(item.GetID()), data)
	})
}

// Pop retrieves and marks the next pending item as processing
func (gq *GenericQueue[T]) Pop() *T {
	items := gq.PopN(1)
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

// PopN retrieves and marks up to n pending items as processing
func (gq *GenericQueue[T]) PopN(n int) []T {
	if n <= 0 {
		return []T{}
	}

	var items []T

	err := gq.queue.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil && len(items) < n; key, value = cursor.Next() {
			var queueItem T
			if err := json.Unmarshal(value, &queueItem); err != nil {
				log.Printf("Error unmarshaling item %s: %v", string(key), err)
				continue
			}

			if queueItem.GetStatus() == StatusPending {
				// Set status to processing
				queueItem.SetStatus(StatusProcessing)

				// Save the updated item back to the database
				updatedData, err := json.Marshal(queueItem)
				if err != nil {
					return fmt.Errorf("failed to marshal updated item: %w", err)
				}

				if err := bucket.Put(key, updatedData); err != nil {
					return fmt.Errorf("failed to update item status: %w", err)
				}

				items = append(items, queueItem)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error getting pending items: %v", err)
		return []T{}
	}

	return items
}

func (gq *GenericQueue[T]) UpdateStatus(id string, status Status) error {
	return gq.queue.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		// Get the existing item
		data := bucket.Get([]byte(id))
		if data == nil {
			return ErrItemNotFound
		}

		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		// Set status on the item
		item.SetStatus(status)

		// Save back to database
		updatedData, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal updated item: %w", err)
		}

		return bucket.Put([]byte(id), updatedData)
	})
}

func (gq *GenericQueue[T]) All() []T {
	var items []T

	err := gq.queue.Read(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var item T
			if err := json.Unmarshal(value, &item); err != nil {
				log.Printf("Error unmarshaling item %s: %v", string(key), err)
				continue
			}
			items = append(items, item)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error getting all items: %v", err)
		return []T{}
	}

	return items
}

type QueueStats struct {
	Total     int
	Pending   int
	Completed int
}

// getStats returns total, pending, and completed item counts
func (gq *GenericQueue[T]) getStats() (QueueStats, error) {
	var total, pending, completed int

	err := gq.queue.Read(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var item T
			if err := json.Unmarshal(value, &item); err != nil {
				log.Printf("Error unmarshaling item %s for stats: %v", string(key), err)
				continue
			}

			total++
			switch item.GetStatus() {
			case StatusPending:
				pending++
			case StatusSuccess, StatusFailed:
				completed++
			}
		}
		return nil
	})

	return QueueStats{
		Total:     total,
		Pending:   pending,
		Completed: completed,
	}, err
}

func (gq *GenericQueue[T]) GetPendingCount() (int, error) {
	stats, err := gq.getStats()
	if err != nil {
		return 0, err
	}

	return stats.Pending, err
}

func (gq *GenericQueue[T]) Get(id string) (*T, error) {
	var item *T

	err := gq.queue.Read(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return ErrItemNotFound
		}

		var queueItem T
		if err := json.Unmarshal(data, &queueItem); err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		item = &queueItem
		return nil
	})

	return item, err
}

func (gq *GenericQueue[T]) Clear() error {
	return gq.queue.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(gq.bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}

		cursor := bucket.Cursor()
		for key, _ := cursor.First(); key != nil; key, _ = cursor.Next() {
			if err := bucket.Delete(key); err != nil {
				log.Printf("Error deleting %s key %s: %v", gq.bucketName, string(key), err)
			}
		}

		return nil
	})
}
