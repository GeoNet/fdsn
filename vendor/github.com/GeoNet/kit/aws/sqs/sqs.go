// Package sqs is for messaging with AWS SQS.
package sqs

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Raw struct {
	Body          string
	ReceiptHandle string
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
	cfg, err := config.LoadDefaultConfig(context.TODO())
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
	input := sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		VisibilityTimeout:   visibilityTimeout,
		WaitTimeSeconds:     20,
	}

	for {
		r, err := s.client.ReceiveMessage(context.TODO(), &input)
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
			}
			return m, nil
		case len(r.Messages) > 1:
			return Raw{}, fmt.Errorf("received more than 1 message: %d", len(r.Messages))
		}
	}
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
