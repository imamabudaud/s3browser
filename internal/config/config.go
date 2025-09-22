package config

import (
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	S3 struct {
		AccessKey string `koanf:"access_key"`
		SecretKey string `koanf:"secret_key"`
		Region    string `koanf:"region"`
		Endpoint  string `koanf:"endpoint"`
		Bucket    string `koanf:"bucket"`
	} `koanf:"s3"`

	JWT struct {
		Secret string `koanf:"secret"`
	} `koanf:"jwt"`

	Queue struct {
		UploadInterval      string `koanf:"upload_interval"`
		MaxConcurrentUpload int    `koanf:"max_concurrent_upload"`
		SkipTLSVerify       bool   `koanf:"skip_tls_verify"`

		PublishInterval      string `koanf:"publish_interval"`
		MaxConcurrentPublish int    `koanf:"max_concurrent_publish"`

		UnpublishInterval      string `koanf:"unpublish_interval"`
		MaxConcurrentUnpublish int    `koanf:"max_concurrent_unpublish"`

		DeleteInterval      string `koanf:"delete_interval"`
		MaxConcurrentDelete int    `koanf:"max_concurrent_delete"`
	} `koanf:"queue"`

	Users []struct {
		Username string `koanf:"username"`
		Password string `koanf:"password"`
		Role     string `koanf:"role"`
	} `koanf:"users"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider("config.yml"), yaml.Parser()); err != nil {
		return nil, err
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, err
	}

	if config.JWT.Secret == "" {
		config.JWT.Secret = "CnInvBKME3bcl9NGZI7EKgu8sgxufzZhj342tBvL3KM2tZnpohLfpMptW1Pj0hSl"
	}

	// Set defaults for queue configuration
	if config.Queue.UploadInterval == "" {
		config.Queue.UploadInterval = "10s"
	}
	if config.Queue.MaxConcurrentUpload < 1 {
		config.Queue.MaxConcurrentUpload = 3
	}
	if config.Queue.PublishInterval == "" {
		config.Queue.PublishInterval = "10s"
	}
	if config.Queue.MaxConcurrentPublish < 1 {
		config.Queue.MaxConcurrentPublish = 3
	}
	if config.Queue.UnpublishInterval == "" {
		config.Queue.UnpublishInterval = "10s"
	}
	if config.Queue.MaxConcurrentUnpublish < 1 {
		config.Queue.MaxConcurrentUnpublish = 3
	}
	if config.Queue.DeleteInterval == "" {
		config.Queue.DeleteInterval = "10s"
	}
	if config.Queue.MaxConcurrentDelete < 1 {
		config.Queue.MaxConcurrentDelete = 3
	}

	return &config, nil
}

// GetUploadInterval returns the upload interval as a time.Duration
func (c *Config) GetUploadInterval() (time.Duration, error) {
	return time.ParseDuration(c.Queue.UploadInterval)
}
