package queueupload

import (
	"s3browser/internal/queue"
)

// Item is an alias for queue.UploadItem for backward compatibility
type Item = queue.UploadItem

const bucketName = "queue_upload"
const dbPath = "data/queue.db"

// Queue wraps the generic queue implementation
type Queue struct {
	*queue.GenericQueue[*queue.UploadItem]
}

// New creates a new upload queue instance
func New() (*Queue, error) {
	genericQueue, err := queue.NewGenericQueue[*queue.UploadItem](dbPath, bucketName)
	if err != nil {
		return nil, err
	}

	return &Queue{GenericQueue: genericQueue}, nil
}

// Push adds a new item to the queue
func (q *Queue) Push(item Item) error {
	return q.GenericQueue.Push(&item)
}

// Pop retrieves and marks the next pending item as processing
func (q *Queue) Pop() *Item {
	result := q.GenericQueue.Pop()
	if result == nil {
		return nil
	}
	return (*Item)(*result)
}

// PopN retrieves and marks up to n pending items as processing
func (q *Queue) PopN(n int) []Item {
	items := q.GenericQueue.PopN(n)
	result := make([]Item, len(items))
	for i, item := range items {
		result[i] = *item
	}
	return result
}

// All retrieves all items from the queue
func (q *Queue) All() []Item {
	items := q.GenericQueue.All()
	result := make([]Item, len(items))
	for i, item := range items {
		result[i] = *item
	}
	return result
}

// Get retrieves a specific item by ID
func (q *Queue) Get(id string) (*Item, error) {
	result, err := q.GenericQueue.Get(id)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return (*Item)(*result), nil
}
