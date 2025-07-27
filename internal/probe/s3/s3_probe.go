package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"

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
	ObjectKey string // e.g. "probe-test-file"
	client    *s3.Client
}

// NewWriteProbe creates a WriteProbe with a persistent S3 client
func NewWriteProbe(endpoint, region, accessKey, secretKey, bucket, objectKey string, useSSL bool) *WriteProbe {
	return &WriteProbe{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		UseSSL:    useSSL,
		ObjectKey: objectKey,
	}
}

func (p *WriteProbe) Probe(ctx context.Context) error {
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
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	buf := make([]byte, 100)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &p.Bucket,
		Key:    &p.ObjectKey,
		Body:   bytes.NewReader(buf),
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
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})
		_, err = p.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &p.Bucket,
			Key:    &p.ObjectKey,
			Body:   bytes.NewReader(buf),
		})
	}
	return err
}

type ReadProbe struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	ObjectKey string // e.g. "probe-test-file"
	client    *s3.Client
}

// NewReadProbe creates a ReadProbe with a persistent S3 client
func NewReadProbe(endpoint, region, accessKey, secretKey, bucket, objectKey string, useSSL bool) *ReadProbe {
	return &ReadProbe{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		UseSSL:    useSSL,
		ObjectKey: objectKey,
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
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
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
		p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
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
