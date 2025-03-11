package client_file

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/teejays/gokutil/scalars"
)

type S3FileClient struct {
	client *s3.S3
	bucket string
}

type S3FileClientRequest struct {
	Endpoint  string // e.g. https://shared-project-dev.nyc3.digitaloceanspaces.com
	Bucket    string
	AccessKey string
	SecretKey string
}

func NewS3FileClient(ctx context.Context, req S3FileClientRequest) (IFileClient, error) {
	if req.Endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}
	if req.AccessKey == "" {
		return nil, fmt.Errorf("accessKey cannot be empty")
	}
	if req.SecretKey == "" {
		return nil, fmt.Errorf("secretKey cannot be empty")
	}
	if req.Bucket == "" {
		return nil, fmt.Errorf("bucket cannot be empty")
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(req.AccessKey, req.SecretKey, ""),
		Endpoint:         aws.String(req.Endpoint),
		Region:           aws.String("us-east-1"), // Digital Ocean requires that we set this to us-east-1 (https://docs.digitalocean.com/products/spaces/how-to/use-aws-sdks)
		S3ForcePathStyle: aws.Bool(false),         // // Configures to use subdomain/virtual calling format. Depending on your version, alternatively use o.UsePathStyle = false
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("creating new S3 session: %w", err)
	}
	s3Client := s3.New(newSession)

	return S3FileClient{
		client: s3Client,
		bucket: req.Bucket,
	}, nil
}

func (c S3FileClient) GetClientType() ClientType {
	return TypeS3
}

func (c S3FileClient) FileUpload(ctx context.Context, file io.Reader) (scalars.ID, error) {
	// Generate a unique key
	objectKey := scalars.NewID()

	// Convert file (io.Reader) to S3 compatible type (io.ReadSeeker)
	buf, err := io.ReadAll(file)
	if err != nil {
		return scalars.ID{}, err
	}
	fileSeeker := bytes.NewReader(buf)

	// Upload the file to S3
	_, err = c.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectKey.String()),
		Body:   fileSeeker,
	})
	if err != nil {
		return scalars.ID{}, fmt.Errorf("uploading file: %w", err)
	}

	return objectKey, nil
}

func (c S3FileClient) FileDataUpload(ctx context.Context, data []byte) (scalars.ID, error) {
	return c.FileUpload(ctx, bytes.NewReader(data))
}

func (c S3FileClient) FileRead(ctx context.Context, id scalars.ID) (io.ReadCloser, error) {
	// Download the file from S3
	resp, err := c.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(id.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading file: %w", err)
	}
	return resp.Body, nil
}

func (c S3FileClient) FileDataRead(ctx context.Context, id scalars.ID) ([]byte, error) {
	// Download the file from S3
	resp, err := c.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(id.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
