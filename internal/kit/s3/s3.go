// Package s3 is for working with AWS S3 buckets.
package s3

import (
	"bytes"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"os"
)

type S3 struct {
	client *s3.S3
}

// New returns an S3 using the default AWS credentials chain.
// This consults (in order) environment vars, config files, ec2 and ecs roles.
// It is an error if the AWS_REGION environment variable is not set.
func New() (S3, error) {
	if os.Getenv("AWS_REGION") == "" {
		return S3{}, errors.New("AWS_REGION is not set")
	}

	sess, err := session.NewSession()
	if err != nil {
		return S3{}, err
	}

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
