package handlers

import (
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"s3browser/internal/s3"
	"time"
)

type S3Service interface {
	ListObjects(prefix string) ([]s3.FileItem, error)
	PutObject(key string, reader io.Reader) error
	MakePublic(key string) error
	MakePrivate(key string) error
	GetObject(key string) (*awss3.GetObjectOutput, error)
	DeleteObject(key string) error
	GetFolderStats(prefix string) (*s3.FolderStats, error)
	GenerateSignedURL(key string, expiration time.Duration) (string, error)
	IsImageFile(key string) bool
	GeneratePublicURL(key string) string
	IsObjectPublic(key string) (bool, string)
}
