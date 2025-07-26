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
}

func (p *WriteProbe) Probe(ctx context.Context) error {
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
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	// Write a 100-byte random file
	buf := make([]byte, 100)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &p.Bucket,
		Key:    &p.ObjectKey,
		Body:   bytes.NewReader(buf),
	})
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
}

func (p *ReadProbe) Probe(ctx context.Context) error {
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
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &p.Bucket,
		Key:    &p.ObjectKey,
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	// Read the file to completion
	_, err = io.ReadAll(out.Body)
	return err
}
