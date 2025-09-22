package queue

// BaseItem provides the common fields and methods for all queue items
type BaseItem struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

// GetID returns the item's ID
func (b BaseItem) GetID() string {
	return b.ID
}

// SetID sets the item's ID
func (b *BaseItem) SetID(id string) {
	b.ID = id
}

// GetStatus returns the item's status
func (b BaseItem) GetStatus() Status {
	return b.Status
}

// SetStatus sets the item's status
func (b *BaseItem) SetStatus(status Status) {
	b.Status = status
}

// UploadItem represents an item in the upload queue
type UploadItem struct {
	BaseItem
	RemoteURL       string `json:"remoteUrl"`
	DestinationName string `json:"destinationName"`
	TargetFolder    string `json:"targetFolder"`
}

// PublishItem represents an item in the publish queue
type PublishItem struct {
	BaseItem
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
}

// UnpublishItem represents an item in the unpublish queue
type UnpublishItem struct {
	BaseItem
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
}

// DeleteItem represents an item in the delete queue
type DeleteItem struct {
	BaseItem
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
}
