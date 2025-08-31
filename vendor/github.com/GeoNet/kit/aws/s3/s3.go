// Package s3 is for working with AWS S3 buckets.
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3 struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

type Meta map[string]string

// New returns an S3 struct which wraps an S3 client using the default AWS credentials chain.
// This consults (in order) environment vars, config files, EC2 and ECS roles.
// It is an error if the AWS_REGION environment variable is not set.
// Requests with recoverable errors will be retried with the default retrier.
func New() (S3, error) {
	cfg, err := getConfig()
	if err != nil {
		return S3{}, err
	}
	return S3{client: s3.NewFromConfig(cfg)}, nil
}

// NewWithMaxRetries returns the same as New(), but with the
// back off set to up to maxRetries times.
func NewWithMaxRetries(maxRetries int) (S3, error) {
	cfg, err := getConfig()
	if err != nil {
		return S3{}, err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.Retryer = retry.AddWithMaxAttempts(options.Retryer, maxRetries)
	})
	return S3{client: client}, nil
}

// NewWithOptions returns the same as New(), but with the additional option functions
// applied.
func NewWithOptions(optFns ...func(*s3.Options)) (S3, error) {
	cfg, err := getConfig()
	if err != nil {
		return S3{}, err
	}
	client := s3.NewFromConfig(cfg, optFns...)
	return S3{client: client}, nil
}

// AddUploader creates an s3manager uploader and sets it to the S3 struct's
// uploader field. This can be used for streaming uploading.
func (s3 *S3) AddUploader() error {
	if !s3.Ready() {
		return errors.New("S3 client needs to be initialised to add an uploader")
	}
	s3.uploader = manager.NewUploader(s3.client)
	return nil
}

// AddDownloader creates an s3manager downloader and sets it to
// the S3 struct's downloader field. This can be used for downloading
// objects in concurrent chunks.
func (s3 *S3) AddDownloader() error {
	if !s3.Ready() {
		return errors.New("S3 client needs to be initialised to add a downloader")
	}
	s3.downloader = manager.NewDownloader(s3.client)
	return nil
}

// getConfig returns the default AWS Config struct.
func getConfig() (aws.Config, error) {
	if os.Getenv("AWS_REGION") == "" {
		return aws.Config{}, errors.New("AWS_REGION is not set")
	}

	var cfg aws.Config
	var err error

	if awsEndpoint := os.Getenv("AWS_ENDPOINT_URL"); awsEndpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               awsEndpoint,
				HostnameImmutable: true,
			}, nil
		})

		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithEndpointResolverWithOptions(customResolver))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

// Ready returns whether the S3 client has been initialised.
func (s *S3) Ready() bool {
	return s.client != nil
}

// Client returns the underlying S3 client.
func (s *S3) Client() *s3.Client {
	return s.client
}

// Get gets the object referred to by key and version from bucket and writes it into b.
// Version can be empty.
func (s *S3) Get(bucket, key, version string, b *bytes.Buffer) error {
	input := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	result, err := s.client.GetObject(context.TODO(), &input)
	if err != nil {
		return err
	}
	defer result.Body.Close()

	_, err = b.ReadFrom(result.Body)

	return err
}

// GetByteRange gets the specified byte range of an object referred to by key and version
// from bucket and writes it into b. Version can be empty.
// See https://www.rfc-editor.org/rfc/rfc9110.html#name-byte-ranges for examples
func (s *S3) GetByteRange(bucket, key, version, byteRange string, b *bytes.Buffer) error {
	input := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
		Range:  aws.String(byteRange),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	result, err := s.client.GetObject(context.TODO(), &input)
	if err != nil {
		return err
	}
	defer result.Body.Close()

	_, err = b.ReadFrom(result.Body)

	return err
}

// GetWithLastModified behaves the same as Get(), but also returns the time that
// the object was last modified.
func (s *S3) GetWithLastModified(bucket, key, version string, b *bytes.Buffer) (time.Time, error) {
	input := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	result, err := s.client.GetObject(context.TODO(), &input)
	if err != nil {
		return time.Time{}, err
	}
	defer result.Body.Close()

	_, err = b.ReadFrom(result.Body)

	return aws.ToTime(result.LastModified), err
}

// LastModified returns the time that the specified object was last modified.
func (s *S3) LastModified(bucket, key, version string) (time.Time, error) {
	input := s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}
	result, err := s.client.GetObject(context.TODO(), &input)
	if err != nil {
		return time.Time{}, err
	}
	defer result.Body.Close()

	return aws.ToTime(result.LastModified), nil
}

// GetMeta returns the metadata for an object. Version can be empty.
func (s *S3) GetMeta(bucket, key, version string) (Meta, error) {
	input := s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}

	res, err := s.client.HeadObject(context.TODO(), &input)
	if err != nil {
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				if ae.ErrorCode() == "NotFound" {
					return Meta{}, nil
				}
			}
			return Meta{}, err
		}
	}

	return res.Metadata, nil
}

// GetContentSize returns the content length and last modified time of the specified key
func (s *S3) GetContentSizeTime(bucket, key string) (int64, time.Time, error) {
	var size int64
	var mt time.Time
	input := s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	o, err := s.client.HeadObject(context.TODO(), &input)
	if err != nil {
		return size, mt, err
	}
	return aws.ToInt64(o.ContentLength), aws.ToTime(o.LastModified), nil
}

// Put puts the object in bucket using specified key.
func (s *S3) Put(bucket, key string, object []byte) error {
	input := s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(object),
	}

	_, err := s.client.PutObject(context.TODO(), &input)
	return err
}

// Put puts the object in bucket with metadata using specified key.
func (s *S3) PutWithMetadata(bucket, key string, object []byte, metadata Meta) error {
	input := s3.PutObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		Body:     bytes.NewReader(object),
		Metadata: metadata,
	}

	_, err := s.client.PutObject(context.TODO(), &input)
	return err
}

// CheckBucket checks if the given S3 bucket exists and is accessible.
func (s *S3) CheckBucket(bucket string) error {
	_, err := s.client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	return err
}

// Exists checks if an object for key already exists in the bucket.
func (s *S3) Exists(bucket, key string) (bool, error) {

	input := s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(context.TODO(), &input)
	if err == nil {
		return true, nil
	}

	var ae smithy.APIError
	if errors.As(err, &ae) {
		if ae.ErrorCode() == "NotFound" {
			return false, nil
		}
	}
	return false, err
}

// List returns a list of object keys that match the provided prefix.
// It will not return more than the specified max number of keys.
// Keys are in alphabetical order.
func (s *S3) List(bucket, prefix string, max int32) ([]string, error) {

	objects, err := s.ListObjects(bucket, prefix, max)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)

	for _, o := range objects {
		result = append(result, aws.ToString(o.Key))
	}
	return result, nil
}

// ListAll returns a list of ALL object keys that match the provided prefix.
// Keys are in alphabetical order.
func (s *S3) ListAll(bucket, prefix string) ([]string, error) {

	objects, err := s.ListAllObjects(bucket, prefix)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)

	for _, o := range objects {
		result = append(result, aws.ToString(o.Key))
	}
	return result, nil
}

// Returns whether there is an object in bucket with specified prefix.
func (s *S3) PrefixExists(bucket, prefix string) (bool, error) {
	maxKeys := int32(1)
	input := s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}
	out, err := s.client.ListObjectsV2(context.TODO(), &input)
	if err != nil {
		return false, err
	}
	if len(out.Contents) > 0 {
		return true, nil
	}
	return false, nil
}

// ListCommonPrefixes returns a list of ALL common prefixes (no 1000 limit).
func (s *S3) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {

	result := make([]string, 0)

	var continuationToken *string

	for {
		input := s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			Delimiter:         aws.String(delimiter),
			ContinuationToken: continuationToken,
		}

		out, err := s.client.ListObjectsV2(context.TODO(), &input)
		if err != nil {
			return nil, err
		}
		for _, o := range out.CommonPrefixes {
			result = append(result, aws.ToString(o.Prefix))
		}
		// When result is not truncated, it means all common prefixes have been found.
		if !(*out.IsTruncated) {
			return result, nil
		}
		continuationToken = out.NextContinuationToken
	}
}

// ListObjects returns a list of objects that match the provided prefix.
// It will not return more than the specified max number of keys.
func (s *S3) ListObjects(bucket, prefix string, max int32) ([]types.Object, error) {
	input := s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &max,
	}

	out, err := s.client.ListObjectsV2(context.TODO(), &input)
	if err != nil {
		return nil, err
	}

	return out.Contents, nil
}

// ListAllObjects returns a list of ALL objects that match the provided prefix.
// Keys are in alphabetical order.
func (s *S3) ListAllObjects(bucket, prefix string) ([]types.Object, error) {

	result := make([]types.Object, 0)

	var continuationToken *string

	for {
		input := s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}

		out, err := s.client.ListObjectsV2(context.TODO(), &input)
		if err != nil {
			return nil, err
		}
		result = append(result, out.Contents...)

		// When result is not truncated, it means all matching keys have been found.
		if !(*out.IsTruncated) {
			return result, nil
		}
		continuationToken = out.NextContinuationToken
	}
}

// ListAllObjectsConcurrently returns a list of ALL objects that match the provided prefixes.
// Keys are NOT in alphabetical order.
func (s *S3) ListAllObjectsConcurrently(bucket string, prefixes []string) ([]types.Object, error) {

	type work struct {
		index  int // Used to retain order.
		bucket string
		prefix string
		result []types.Object
		err    error
	}

	input := make(chan work, len(prefixes))
	output := make(chan work)

	workerCount := 20
	var wg sync.WaitGroup
	wg.Add(workerCount)

	// Workers take work from the input channel. work contains a prefix to List from S3.
	// The result of the list is set, then sent to the output channel.
	worker := func(s *S3, input <-chan work, output chan<- work, wg *sync.WaitGroup) {
		defer wg.Done()

		for w := range input {
			w.result, w.err = s.ListAllObjects(w.bucket, w.prefix)
			output <- w
		}
	}
	// Create workers
	for i := 0; i < workerCount; i++ {
		go worker(s, input, output, &wg)
	}
	// Send prefixes to list to the workers.
	for i, prefix := range prefixes {
		input <- work{index: i, bucket: bucket, prefix: prefix}
	}
	close(input)

	go func() {
		wg.Wait()
		close(output)
	}()

	// Read results and errors from output channel.
	results := make([][]types.Object, len(prefixes))
	errorList := make([]error, 0)
	for w := range output {
		if w.err != nil {
			errorList = append(errorList, w.err)
		}
		results[w.index] = w.result
	}

	// If errors found, concatenate and return as error.
	if len(errorList) > 0 {
		var errorMessage string
		for _, e := range errorList {
			errorMessage += e.Error() + "\n"
		}
		return nil, errors.New(errorMessage)
	}

	// Append results into single slice before returning.
	finalResult := make([]types.Object, 0)
	for _, r := range results {
		finalResult = append(finalResult, r...)
	}
	return finalResult, nil
}

// PutStream puts the data stream to key in bucket.
func (s *S3) PutStream(bucket, key string, reader io.ReadCloser) error {
	defer reader.Close()

	if s.uploader == nil {
		return errors.New("error uploading to s3, uploader not initialised")
	}
	input := s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	_, err := s.uploader.Upload(context.TODO(), &input)
	if err != nil {
		return fmt.Errorf("error uploading to s3 for key %s, error: %s", key, err.Error())
	}
	return nil
}

// Download uses the downloader to download file from bucket.
// File is split up into parts and downloaded concurrently into an os.File,
// so is useful for getting large files. Returns number of bytes downloaded.
func (s *S3) Download(bucket, key string, f *os.File) (int64, error) {
	if s.downloader == nil {
		return 0, errors.New("error downloading from S3, downloader not initialised")
	}
	input := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	numBytes, err := s.downloader.Download(context.TODO(), f, &input)
	if err != nil {
		return 0, err
	}
	return numBytes, nil
}

// Delete deletes an object from a bucket.
func (s *S3) Delete(bucket, key string) error {
	input := s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(context.TODO(), &input)
	return err
}

// Copy copies from the source to the bucket with key as the new name.
// source should include the bucket name eg: "mybucket/objectkey.pdf"
func (s *S3) Copy(bucket, key, source string) error {
	input := s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		CopySource: aws.String(source),
	}
	_, err := s.client.CopyObject(context.TODO(), &input)

	return err
}

// CreateBucket creates a bucket.
func (s *S3) CreateBucket(bucket string) error {
	config := types.CreateBucketConfiguration{
		LocationConstraint: types.BucketLocationConstraint(s.client.Options().Region),
	}

	input := s3.CreateBucketInput{
		Bucket:                    aws.String(bucket),
		CreateBucketConfiguration: &config,
	}
	_, err := s.client.CreateBucket(context.TODO(), &input)

	return err
}

// DeleteBucket deletes a bucket.
func (s *S3) DeleteBucket(bucket string) error {
	input := s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}
	_, err := s.client.DeleteBucket(context.TODO(), &input)

	return err
}
