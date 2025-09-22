package s3

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Service struct {
	client     *s3.Client
	bucketName string
	endpoint   string
}

type FileItem struct {
	Name         string
	Key          string
	Size         int64
	LastModified string
	IsDir        bool
	IsPublic     bool
	PublicURL    string
}

type FolderStats struct {
	FileCount  int64  `json:"fileCount"`
	TotalSize  int64  `json:"totalSize"`
	FolderPath string `json:"folderPath"`
}

func NewS3Service(accessKey, secretKey, region, endpoint, bucketName string) (*S3Service, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	// Override endpoint if provided
	if endpoint != "" {
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Service{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
	}, nil
}

func (s *S3Service) ListObjects(prefix string) ([]FileItem, error) {
	ctx := context.Background()
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	var files []FileItem
	seenDirs := make(map[string]bool)

	for _, obj := range result.Contents {
		key := *obj.Key

		// Skip the prefix itself if it's a folder marker
		if key == prefix {
			continue
		}

		// Get the relative path from the prefix
		relativePath := strings.TrimPrefix(key, prefix)

		// Split by "/" to get the first part (immediate child)
		parts := strings.Split(relativePath, "/")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		// Get the immediate child name
		childName := parts[0]

		// Check if this is a direct child (not nested deeper)
		if len(parts) == 1 {
			// This is a direct file
			isPublic := false
			publicURL := ""

			// Check if file is public by examining ACL
			acl, err := s.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{
				Bucket: aws.String(s.bucketName),
				Key:    aws.String(key),
			})
			if err == nil {
				fmt.Printf("DEBUG: ACL grants for %s: %+v\n", key, acl.Grants)
				for _, grant := range acl.Grants {
					if grant.Grantee != nil && grant.Grantee.URI != nil {
						uri := *grant.Grantee.URI
						fmt.Printf("DEBUG: Grant URI for %s: %s\n", key, uri)
						// Check for both old and new URI formats
						if uri == "http://acs.amazonaws.com/groups/global/AllUsers" ||
							uri == "https://acs.amazonaws.com/groups/global/AllUsers" {
							isPublic = true
							// Construct proper public URL based on endpoint
							// For DigitalOcean Spaces: https://{bucket}.{region}.digitaloceanspaces.com/{key}
							// For AWS S3: https://{bucket}.s3.{region}.amazonaws.com/{key}
							if strings.Contains(s.endpoint, "digitaloceanspaces.com") {
								// Extract region from endpoint (e.g., sgp1 from https://sgp1.digitaloceanspaces.com)
								parts := strings.Split(s.endpoint, "://")
								if len(parts) > 1 {
									hostParts := strings.Split(parts[1], ".")
									if len(hostParts) > 0 {
										region := hostParts[0]
										publicURL = fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", s.bucketName, region, key)
									} else {
										publicURL = fmt.Sprintf("https://%s.digitaloceanspaces.com/%s", s.bucketName, key)
									}
								} else {
									publicURL = fmt.Sprintf("https://%s.digitaloceanspaces.com/%s", s.bucketName, key)
								}
							} else {
								// Default to bucket.s3.region.amazonaws.com format
								publicURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucketName, key)
							}
							fmt.Printf("DEBUG: File %s is public, URL: %s\n", key, publicURL)
							break
						}
					}
				}
			} else {
				fmt.Printf("DEBUG: Error getting ACL for %s: %v\n", key, err)
			}
			// If ACL check fails, assume private (isPublic remains false)

			files = append(files, FileItem{
				Name:         childName,
				Key:          key,
				Size:         *obj.Size,
				LastModified: obj.LastModified.Format("2006-01-02 15:04:05"),
				IsDir:        false,
				IsPublic:     isPublic,
				PublicURL:    publicURL,
			})
		} else if len(parts) == 2 && parts[1] == "" {
			// This is a direct folder (ends with /)
			folderKey := prefix + childName + "/"
			if !seenDirs[folderKey] {
				files = append(files, FileItem{
					Name:         childName,
					Key:          folderKey,
					IsDir:        true,
					LastModified: obj.LastModified.Format("2006-01-02 15:04:05"),
				})
				seenDirs[folderKey] = true
			}
		}
	}

	return files, nil
}

func (s *S3Service) PutObject(key string, reader io.Reader) error {
	ctx := context.Background()

	// Detect content type from file extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(path.Ext(key))
	if detectedType := mime.TypeByExtension(ext); detectedType != "" {
		contentType = detectedType
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String(contentType),
	})
	return err
}

func (s *S3Service) MakePublic(key string) error {
	ctx := context.Background()
	_, err := s.client.PutObjectAcl(ctx, &s3.PutObjectAclInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		ACL:    types.ObjectCannedACLPublicRead,
	})
	return err
}

func (s *S3Service) MakePrivate(key string) error {
	ctx := context.Background()
	_, err := s.client.PutObjectAcl(ctx, &s3.PutObjectAclInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		ACL:    types.ObjectCannedACLPrivate,
	})
	return err
}

func (s *S3Service) GetObject(key string) (*s3.GetObjectOutput, error) {
	ctx := context.Background()
	return s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
}

func (s *S3Service) DeleteObject(key string) error {
	ctx := context.Background()
	// For folders (keys ending with /), we need to delete all objects with that prefix
	if strings.HasSuffix(key, "/") {
		return s.deleteFolder(key)
	}

	// For regular files, just delete the object
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Service) deleteFolder(prefix string) error {
	ctx := context.Background()
	// List all objects with this prefix
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return err
	}

	// Delete all objects in the folder
	for _, obj := range result.Contents {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    obj.Key,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFolderStats returns statistics for a folder (file count and total size)
func (s *S3Service) GetFolderStats(prefix string) (*FolderStats, error) {
	ctx := context.Background()
	// Ensure prefix ends with / for folder
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}

	var fileCount int64
	var totalSize int64

	// Paginate through all objects using the new v2 paginator
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			// Skip the folder marker itself (object with key ending in /)
			if strings.HasSuffix(*obj.Key, "/") && *obj.Key == prefix {
				continue
			}

			// Count files (not folders)
			if !strings.HasSuffix(*obj.Key, "/") {
				fileCount++
				if obj.Size != nil {
					totalSize += *obj.Size
				}
			}
		}
	}

	return &FolderStats{
		FileCount:  fileCount,
		TotalSize:  totalSize,
		FolderPath: prefix,
	}, nil
}

// GenerateSignedURL generates a presigned URL for the given object key
func (s *S3Service) GenerateSignedURL(key string, expiration time.Duration) (string, error) {
	ctx := context.Background()

	// Create a presigner
	presigner := s3.NewPresignClient(s.client)

	// Create the request
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", err
	}

	return request.URL, nil
}

// IsImageFile checks if the given file key represents an image file
func (s *S3Service) IsImageFile(key string) bool {
	ext := strings.ToLower(path.Ext(key))
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".tiff", ".ico"}

	for _, imgExt := range imageExtensions {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// GeneratePublicURL generates a public URL for the given object key
// For DigitalOcean Spaces: https://{bucket}.{region}.digitaloceanspaces.com/{key}
// For AWS S3: https://{bucket}.s3.{region}.amazonaws.com/{key}
func (s *S3Service) GeneratePublicURL(key string) string {
	if strings.Contains(s.endpoint, "digitaloceanspaces.com") {
		// Extract region from endpoint (e.g., sgp1 from https://sgp1.digitaloceanspaces.com)
		parts := strings.Split(s.endpoint, "://")
		if len(parts) > 1 {
			hostParts := strings.Split(parts[1], ".")
			if len(hostParts) > 0 {
				region := hostParts[0]
				return fmt.Sprintf("https://%s.%s.digitaloceanspaces.com/%s", s.bucketName, region, key)
			} else {
				return fmt.Sprintf("https://%s.digitaloceanspaces.com/%s", s.bucketName, key)
			}
		} else {
			return fmt.Sprintf("https://%s.digitaloceanspaces.com/%s", s.bucketName, key)
		}
	} else {
		// Default to bucket.s3.region.amazonaws.com format
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucketName, key)
	}
}

// IsObjectPublic checks if the given object is public by examining its ACL
func (s *S3Service) IsObjectPublic(key string) (bool, string) {
	ctx := context.Background()

	acl, err := s.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, ""
	}

	for _, grant := range acl.Grants {
		if grant.Grantee != nil && grant.Grantee.URI != nil {
			uri := *grant.Grantee.URI
			if uri == "http://acs.amazonaws.com/groups/global/AllUsers" ||
				uri == "https://acs.amazonaws.com/groups/global/AllUsers" {
				return true, s.GeneratePublicURL(key)
			}
		}
	}

	return false, ""
}
