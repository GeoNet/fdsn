// Package sqs is for messaging with AWS SQS.
package sqs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	smithy "github.com/aws/smithy-go"
)

type Raw struct {
	Body          string
	ReceiptHandle string
	Attributes    map[string]string
}

type SQS struct {
	client *sqs.Client
}

// New returns an SQS struct which wraps an SQS client using the default AWS credentials chain.
// This consults (in order) environment vars, config files, EC2 and ECS roles.
// It is an error if the AWS_REGION environment variable is not set.
// Requests with recoverable errors will be retried with the default retrier.
func New() (SQS, error) {
	cfg, err := getConfig()
	if err != nil {
		return SQS{}, err
	}
	return SQS{client: sqs.NewFromConfig(cfg)}, nil
}

// NewWithMaxRetries returns the same as New(), but with the
// back off set to up to maxRetries times.
func NewWithMaxRetries(maxRetries int) (SQS, error) {
	cfg, err := getConfig()
	if err != nil {
		return SQS{}, err
	}
	client := sqs.NewFromConfig(cfg, func(options *sqs.Options) {
		options.Retryer = retry.AddWithMaxAttempts(options.Retryer, maxRetries)
	})

	return SQS{client: client}, nil
}

// getConfig returns the default AWS Config struct.
func getConfig() (aws.Config, error) {
	if os.Getenv("AWS_REGION") == "" {
		return aws.Config{}, errors.New("AWS_REGION is not set")
	}

	var cfg aws.Config
	var err error

	if awsEndpoint := os.Getenv("CUSTOM_AWS_ENDPOINT_URL"); awsEndpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				SigningRegion: region,
				URL:           awsEndpoint,
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

// Ready returns whether the SQS client has been initialised.
func (s *SQS) Ready() bool {
	return s.client != nil
}

// Receive receives a raw message or error from the queue.
// After a successful receive the message will be in flight
// until it is either deleted or the visibility timeout expires
// (at which point it is available for redelivery).
//
// Applications should be able to handle duplicate or out of order messages,
// and should back off on Receive error.
func (s *SQS) Receive(queueURL string, visibilityTimeout int32) (Raw, error) {
	return s.ReceiveWithContext(context.TODO(), queueURL, visibilityTimeout)
}

// ReceiveWithAttributes is the same as Receive except that Queue Attributes can be requested
// to be received with the message.
func (s *SQS) ReceiveWithAttributes(queueURL string, visibilityTimeout int32, attrs []types.QueueAttributeName) (Raw, error) {
	return s.ReceiveWithContextAttributes(context.TODO(), queueURL, visibilityTimeout, attrs)
}

// ReceiveWithContextAttributes by context and Queue Attributes,
// so that system stop signal can be received by the context.
// to receive system stop signal, register the context with signal.NotifyContext before passing in this function,
// when system stop signal is received, an error with message '... context canceled' will be returned
// which can be used to safely stop the system
func (s *SQS) ReceiveWithContextAttributes(ctx context.Context, queueURL string, visibilityTimeout int32, attrs []types.QueueAttributeName) (Raw, error) {
	input := sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   visibilityTimeout,
		WaitTimeSeconds:     20,
		AttributeNames:      attrs,
	}
	return s.receiveMessage(ctx, &input)
}

// receiveMessage is the common code used internally to receive an SQS message based
// on the provided input.
func (s *SQS) receiveMessage(ctx context.Context, input *sqs.ReceiveMessageInput) (Raw, error) {

	for {
		r, err := s.client.ReceiveMessage(ctx, input)
		if err != nil {
			return Raw{}, err
		}

		switch {
		case r == nil || len(r.Messages) == 0:
			// no message received
			continue
		case len(r.Messages) == 1:
			raw := r.Messages[0]

			m := Raw{
				Body:          aws.ToString(raw.Body),
				ReceiptHandle: aws.ToString(raw.ReceiptHandle),
				Attributes:    raw.Attributes,
			}
			return m, nil
		case len(r.Messages) > 1:
			return Raw{}, fmt.Errorf("received more than 1 message: %d", len(r.Messages))
		}
	}
}

// receive with context so that system stop signal can be received,
// to receive system stop signal, register the context with signal.NotifyContext before passing in this function,
// when system stop signal is received, an error with message '... context canceled' will be returned
// which can be used to safely stop the system
func (s *SQS) ReceiveWithContext(ctx context.Context, queueURL string, visibilityTimeout int32) (Raw, error) {
	input := sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   visibilityTimeout,
		WaitTimeSeconds:     20,
	}
	return s.receiveMessage(ctx, &input)
}

// Delete deletes the message referred to by receiptHandle from the queue.
func (s *SQS) Delete(queueURL, receiptHandle string) error {
	params := sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := s.client.DeleteMessage(context.TODO(), &params)

	return err
}

// Send sends the message body to the SQS queue referred to by queueURL.
func (s *SQS) Send(queueURL string, body string) error {
	params := sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	}

	_, err := s.client.SendMessage(context.TODO(), &params)

	return err
}

// SendWithDelay is the same as Send but adds a delay (in seconds) before sending.
func (s *SQS) SendWithDelay(queueURL string, body string, delay int32) error {
	params := sqs.SendMessageInput{
		QueueUrl:     aws.String(queueURL),
		MessageBody:  aws.String(body),
		DelaySeconds: delay,
	}

	_, err := s.client.SendMessage(context.TODO(), &params)

	return err
}

// SendFifoMessage puts a message onto the given AWS SQS queue.
func (s *SQS) SendFifoMessage(queue, group, dedupe string, msg []byte) (string, error) {
	var id *string
	if dedupe != "" {
		id = aws.String(dedupe)
	}
	params := sqs.SendMessageInput{
		MessageBody:            aws.String(string(msg)),
		QueueUrl:               aws.String(queue),
		MessageGroupId:         aws.String(group),
		MessageDeduplicationId: id,
	}
	output, err := s.client.SendMessage(context.TODO(), &params)
	if err != nil {
		return "", err
	}
	if id := output.MessageId; id != nil {
		return *id, nil
	}
	return "", nil
}

// Leverage the sendbatch api for uploading large numbers of messages
func (s *SQS) SendBatch(ctx context.Context, queueURL string, bodies []string) error {
	if len(bodies) > 11 {
		return errors.New("too many messages to batch")
	}
	var err error
	entries := make([]types.SendMessageBatchRequestEntry, len(bodies))
	for j, body := range bodies {
		entries[j] = types.SendMessageBatchRequestEntry{
			Id:          aws.String(fmt.Sprintf("gamitjob%d", j)),
			MessageBody: aws.String(body),
		}
	}
	_, err = s.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: &queueURL,
	})
	return err
}

func (s *SQS) SendNBatch(ctx context.Context, queueURL string, bodies []string) error {
	var (
		bodiesLen = len(bodies)
		maxlen    = 10
		times     = int(math.Ceil(float64(bodiesLen) / float64(maxlen)))
	)
	for i := 0; i < times; i++ {
		batch_end := maxlen * (i + 1)
		if maxlen*(i+1) > bodiesLen {
			batch_end = bodiesLen
		}
		var bodies_batch = bodies[maxlen*i : batch_end]
		err := s.SendBatch(ctx, queueURL, bodies_batch)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetQueueUrl returns an AWS SQS queue URL given its name.
func (s *SQS) GetQueueUrl(name string) (string, error) {
	params := sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	}
	output, err := s.client.GetQueueUrl(context.TODO(), &params)
	if err != nil {
		return "", err
	}
	if url := output.QueueUrl; url != nil {
		return aws.ToString(url), nil
	}
	return "", nil
}

func Cancelled(err error) bool {
	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		return opErr.Service() == "SQS" && strings.Contains(opErr.Unwrap().Error(), "context canceled")
	}
	return false
}
