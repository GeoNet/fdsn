// Package s3 is for working with AWS S3 buckets.
package s3

import (
	"bytes"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"os"
	"time"
)

type S3 struct {
	client *s3.S3
}

// New returns an S3 using the default AWS credentials chain.
// This consults (in order) environment vars, config files, ec2 and ecs roles.
// It is an error if the AWS_REGION environment variable is not set.
// Requests with recoverable errors will be retried with the default retrier
// with back off up to maxRetries times.
func New(maxRetries int) (S3, error) {
	if os.Getenv("AWS_REGION") == "" {
		return S3{}, errors.New("AWS_REGION is not set")
	}

	sess, err := session.NewSession()
	if err != nil {
		return S3{}, err
	}

	// logging can be made more verbose e.g.,
	//sess.Config.WithLogLevel(aws.LogDebugWithRequestRetries)

	sess.Config.Retryer = client.DefaultRetryer{NumMaxRetries: maxRetries}

	return S3{client: s3.New(sess)}, nil
}

// Get gets the object referred to by key and version from bucket and write is into b.
// version can be zero.
func (s *S3) Get(bucket, key, version string, b *bytes.Buffer) error {
	params := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	}

	if version != "" {
		params.VersionId = aws.String(version)
	}

	result, err := s.client.GetObject(&params)
	if err != nil {
		return err
	}
	defer result.Body.Close()

	_, err = b.ReadFrom(result.Body)

	return err
}

func (s *S3) LastModified(bucket, key, version string) (*time.Time, error) {
	params := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	}

	if version != "" {
		params.VersionId = aws.String(version)
	}

	result, err := s.client.GetObject(&params)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	return result.LastModified, nil
}
