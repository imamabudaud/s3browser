package handlers

import (
	"s3browser/internal/auth"
	"s3browser/internal/queue/queuedelete"
	"s3browser/internal/queue/queuepublish"
	"s3browser/internal/queue/queueunpublish"
	"s3browser/internal/queue/queueupload"
)

type Handlers struct {
	s3Service      S3Service
	queueUpload    *queueupload.Queue
	publishQueue   *queuepublish.Queue
	unpublishQueue *queueunpublish.Queue
	deleteQueue    *queuedelete.Queue
	jwtService     *auth.JWTService
}

func NewHandlers(s3Service S3Service,
	uploadQueue *queueupload.Queue,
	publishQueue *queuepublish.Queue,
	unpublishQueue *queueunpublish.Queue,
	deleteQueue *queuedelete.Queue,
	jwtService *auth.JWTService,
) *Handlers {

	return &Handlers{
		s3Service:      s3Service,
		queueUpload:    uploadQueue,
		publishQueue:   publishQueue,
		unpublishQueue: unpublishQueue,
		deleteQueue:    deleteQueue,
		jwtService:     jwtService,
	}
}
