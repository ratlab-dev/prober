package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type WriteProbe struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	ObjectKey string        // e.g. "probe-test-file" (used for read probe only)
	Timeout   time.Duration // Timeout in seconds for S3 operations
	client    *s3.Client
}

// NewWriteProbe creates a WriteProbe with a persistent S3 client
func NewWriteProbe(endpoint, region, accessKey, secretKey, bucket, objectKey string, useSSL bool, timeout time.Duration) *WriteProbe {
	return &WriteProbe{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		UseSSL:    useSSL,
		ObjectKey: objectKey,
		Timeout:   timeout,
	}
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	key := fmt.Sprintf("probe_file_%s", RandString(12))
	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][S3][%s] Writing key: %s", p.Bucket, key)
	}
	if p.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.AccessKey, p.SecretKey, "")),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: p.Endpoint, HostnameImmutable: true, SigningRegion: p.Region}, nil
				},
			)),
		)
		if err != nil {
			return err
		}
		httpClient := &http.Client{
			Timeout: p.Timeout,
		}
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.RetryMaxAttempts = 1
			o.UsePathStyle = true
			o.HTTPClient = httpClient
		})
	}
	buf := make([]byte, 100)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:  &p.Bucket,
		Key:     &key,
		Body:    bytes.NewReader(buf),
		Expires: aws.Time(time.Now().Add(30 * time.Minute)),
	})
	return err
}

// RandString generates a random alphanumeric string of given length
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type ReadProbe struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	ObjectKey string        // e.g. "probe-test-file"
	Timeout   time.Duration // Timeout in seconds for S3 operations
	client    *s3.Client
}

// NewReadProbe creates a ReadProbe with a persistent S3 client
func NewReadProbe(endpoint, region, accessKey, secretKey, bucket, objectKey string, useSSL bool, timeout time.Duration) *ReadProbe {
	return &ReadProbe{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		UseSSL:    useSSL,
		ObjectKey: objectKey,
		Timeout:   timeout,
	}
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	if p.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.AccessKey, p.SecretKey, "")),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: p.Endpoint, HostnameImmutable: true, SigningRegion: p.Region}, nil
				},
			)),
		)
		if err != nil {
			return err
		}
		httpClient := &http.Client{
			Timeout: time.Duration(p.Timeout) * time.Second,
		}
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.HTTPClient = httpClient
			o.RetryMaxAttempts = 1
		})
	}
	out, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &p.Bucket,
		Key:    &p.ObjectKey,
	})
	if err != nil {
		// Try to recover by recreating the client once
		cfg, cfgErr := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.AccessKey, p.SecretKey, "")),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: p.Endpoint, HostnameImmutable: true, SigningRegion: p.Region}, nil
				},
			)),
		)
		if cfgErr != nil {
			return err // return original error if recovery fails
		}
		httpClient := &http.Client{
			Timeout: time.Duration(p.Timeout) * time.Second,
		}
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.HTTPClient = httpClient
		})
		out, err = p.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &p.Bucket,
			Key:    &p.ObjectKey,
		})
		if err != nil {
			return err
		}
	}
	defer out.Body.Close()
	_, err = io.ReadAll(out.Body)
	return err
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("S3ReadProbe | Endpoint: %s | Bucket: %s | Region: %s", p.Endpoint, p.Bucket, p.Region)
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("S3WriteProbe | Endpoint: %s | Bucket: %s | Region: %s", p.Endpoint, p.Bucket, p.Region)
}
