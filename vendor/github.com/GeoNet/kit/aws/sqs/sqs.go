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

// specific error to return when no messages are received from the queue
var ErrNoMessages = errors.New("no messages received from queue")

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

	if awsEndpoint := os.Getenv("AWS_ENDPOINT_URL"); awsEndpoint != "" {
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

// Client returns the underlying SQS client.
func (s *SQS) Client() *sqs.Client {
	return s.client
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
	msgs, err := s.receiveMessages(ctx, &input)
	if err != nil {
		return Raw{}, err
	}
	return msgs[0], err
}

// receiveMessages is the common code used internally to receive an SQS messages based
// on the provided input.
func (s *SQS) receiveMessages(ctx context.Context, input *sqs.ReceiveMessageInput) ([]Raw, error) {
	r, err := s.client.ReceiveMessage(ctx, input)
	if err != nil {
		return []Raw{}, err
	}

	switch {
	case r == nil || len(r.Messages) == 0:
		// no message received
		return []Raw{}, ErrNoMessages

	case len(r.Messages) >= 1:

		messages := make([]Raw, len(r.Messages))
		for i := range r.Messages {
			messages[i] = Raw{
				Body:          aws.ToString(r.Messages[i].Body),
				ReceiptHandle: aws.ToString(r.Messages[i].ReceiptHandle),
				Attributes:    r.Messages[i].Attributes,
			}
		}
		return messages, nil

	default:
		return []Raw{}, fmt.Errorf("received unexpected number of messages: %d", len(r.Messages)) // Probably an impossible case
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
	msgs, err := s.receiveMessages(ctx, &input)
	if err != nil {
		return Raw{}, err
	}
	return msgs[0], err
}

// ReceiveBatch is similar to Receive, however it can return up to 10 messages.
func (s *SQS) ReceiveBatch(ctx context.Context, queueURL string, visibilityTimeout int32) ([]Raw, error) {

	input := sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 10,
		VisibilityTimeout:   visibilityTimeout,
		WaitTimeSeconds:     20,
	}

	msgs, err := s.receiveMessages(ctx, &input)
	if err != nil {
		return []Raw{}, err
	}
	return msgs, nil
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

// SetMessageVisibility sets the visibility timeout for a message.
func (s *SQS) SetMessageVisibility(queueURL, receiptHandle string, visibilityTimeout int32) error {
	params := sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: visibilityTimeout,
	}

	_, err := s.client.ChangeMessageVisibility(context.TODO(), &params)

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

type SendBatchError struct {
	Err  error
	Info []SendBatchErrorEntry
}
type SendBatchErrorEntry struct {
	Entry types.BatchResultErrorEntry
	Index int
}

func (s *SendBatchError) Error() string {
	return fmt.Sprintf("%v: %v messages failed to send", s.Err, len(s.Info))
}
func (s *SendBatchError) Unwrap() error {
	return s.Err
}

type SendNBatchError struct {
	Errors []error
	Info   []SendBatchErrorEntry
}

func (s *SendNBatchError) Error() string {
	var allErrors string
	for _, err := range s.Errors {
		allErrors += fmt.Sprintf("%s,", err.Error())
	}
	allErrors = strings.TrimSuffix(allErrors, ",")
	return fmt.Sprintf("%v error(s) sending batches: %s", len(s.Errors), allErrors)
}

// SendBatch sends up to 10 messages to a given SQS queue with one API call.
// If an error occurs on any or all messages, a SendBatchError is returned that lets
// the caller know the index of the message/s in bodies that failed.
func (s *SQS) SendBatch(ctx context.Context, queueURL string, bodies []string) error {

	var err error
	entries := make([]types.SendMessageBatchRequestEntry, len(bodies))
	for j, body := range bodies {
		entries[j] = types.SendMessageBatchRequestEntry{
			Id:          aws.String(fmt.Sprintf("message-%d", j)),
			MessageBody: aws.String(body),
		}
	}
	output, err := s.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: &queueURL,
	})
	if err != nil {
		info := make([]SendBatchErrorEntry, len(entries))
		for i := range entries {
			info[i] = SendBatchErrorEntry{
				Index: i,
			}
		}
		return &SendBatchError{Err: err, Info: info}
	}
	if len(output.Failed) > 0 {
		info := make([]SendBatchErrorEntry, len(output.Failed))
		for i, entry := range output.Failed {
			for j, msg := range entries {
				if aws.ToString(msg.Id) == aws.ToString(entry.Id) {
					info[i] = SendBatchErrorEntry{
						Entry: entry,
						Index: j,
					}
					break
				}
			}
		}
		return &SendBatchError{Err: errors.New("partial message failure"), Info: info}
	}
	return nil
}

// SendNBatch sends any number of messages to a given SQS queue via a series of SendBatch calls.
// If an error occurs on any or all messages, a SendNBatchError is returned that lets
// the caller know the index of the message/s in bodies that failed.
// Returns the number of API calls to SendBatch made.
func (s *SQS) SendNBatch(ctx context.Context, queueURL string, bodies []string) (int, error) {

	const (
		maxCount = 10
		maxSize  = 262144 // 256 KiB
	)

	allErrors := make([]error, 0)
	allInfo := make([]SendBatchErrorEntry, 0)

	batchesSent := 0

	batch := make([]int, 0)
	totalSize := 0

	sendBatch := func() {
		batchBodies := make([]string, len(batch))

		for i, batchIndex := range batch {
			batchBodies[i] = bodies[batchIndex]
		}

		err := s.SendBatch(ctx, queueURL, batchBodies)
		var sbe *SendBatchError
		if errors.As(err, &sbe) {
			allErrors = append(allErrors, err)

			// Update index so that index refers to the position in given bodies slice.
			for i := range sbe.Info {
				sbe.Info[i].Index = batch[sbe.Info[i].Index]
			}

			allInfo = append(allInfo, sbe.Info...)
		}

		batchesSent++
		batch = batch[:0]
		totalSize = 0
	}

	for i, body := range bodies {

		// Check if any single message is too big
		if len(body) > maxSize {
			allErrors = append(allErrors, errors.New("message too big to send"))
			allInfo = append(allInfo, SendBatchErrorEntry{
				Index: i,
			})
			continue
		}
		// If adding the current message would exceed the batch max size or count, send the current batch.
		if totalSize+len(body) > maxSize || len(batch) == maxCount {
			sendBatch()
		}
		batch = append(batch, i)
		totalSize += len(body)
	}

	if len(batch) > 0 {
		sendBatch()
	}

	if len(allErrors) > 0 {
		return batchesSent, &SendNBatchError{
			Errors: allErrors,
			Info:   allInfo,
		}
	}

	return batchesSent, nil
}

type DeleteBatchError struct {
	Err  error
	Info []DeleteBatchErrorEntry
}

type DeleteBatchErrorEntry struct {
	Entry types.BatchResultErrorEntry
	Index int
}

func (d *DeleteBatchError) Error() string {
	return fmt.Sprintf("%v: %v messages failed to delete", d.Err, len(d.Info))
}

func (d *DeleteBatchError) Unwrap() error {
	return d.Err
}

type DeleteNBatchError struct {
	Errors []error
	Info   []DeleteBatchErrorEntry
}

func (s *DeleteNBatchError) Error() string {
	var allErrors string
	for _, err := range s.Errors {
		allErrors += fmt.Sprintf("%s,", err.Error())
	}
	allErrors = strings.TrimSuffix(allErrors, ",")
	return fmt.Sprintf("%v error(s) deleting batches: %s", len(s.Errors), allErrors)
}

// DeleteBatch deletes up to 10 messages from an SQS queue in a single batch.
// If an error occurs on any or all messages, a DeleteBatchError is returned that lets
// the caller know the indice/s in receiptHandles that failed.
func (s *SQS) DeleteBatch(ctx context.Context, queueURL string, receiptHandles []string) error {
	entries := make([]types.DeleteMessageBatchRequestEntry, len(receiptHandles))
	for i, receipt := range receiptHandles {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(fmt.Sprintf("delete-message-%d", i)),
			ReceiptHandle: aws.String(receipt),
		}
	}

	output, err := s.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: &queueURL,
	})
	if err != nil {
		info := make([]DeleteBatchErrorEntry, len(entries))
		for i := range entries {
			info[i] = DeleteBatchErrorEntry{
				Index: i,
			}
		}
		return &DeleteBatchError{Err: err, Info: info}
	}
	if len(output.Failed) > 0 {
		info := make([]DeleteBatchErrorEntry, len(output.Failed))
		for i, errorEntry := range output.Failed {
			for j, requestEntry := range entries {
				if aws.ToString(requestEntry.Id) == aws.ToString(errorEntry.Id) {
					info[i] = DeleteBatchErrorEntry{
						Entry: errorEntry,
						Index: j,
					}
					break
				}
			}
		}
		return &DeleteBatchError{Info: info}
	}
	return nil
}

// DeleteNBatch deletes any number of messages from a given SQS queue via a series of DeleteBatch calls.
// If an error occurs on any or all messages, a DeleteNBatchError is returned that lets
// the caller know the receipt handles that failed.
// Returns the number of API calls to DeleteBatch made.
func (s *SQS) DeleteNBatch(ctx context.Context, queueURL string, receiptHandles []string) (int, error) {

	var (
		receiptCount = len(receiptHandles)
		maxlen       = 10
		times        = int(math.Ceil(float64(receiptCount) / float64(maxlen)))
	)

	allErrors := make([]error, 0)
	allInfo := make([]DeleteBatchErrorEntry, 0)

	batchesDeleted := 0

	for i := 0; i < times; i++ {
		batch_end := maxlen * (i + 1)
		if maxlen*(i+1) > receiptCount {
			batch_end = receiptCount
		}
		var receipt_batch = receiptHandles[maxlen*i : batch_end]

		indexMap := make(map[int]int, 0)
		count := 0
		for j := maxlen * i; j < batch_end; j++ {
			indexMap[count] = j
			count++
		}

		err := s.DeleteBatch(ctx, queueURL, receipt_batch)
		var dbe *DeleteBatchError
		if errors.As(err, &dbe) {
			allErrors = append(allErrors, err)

			// Update index so that index refers to the position in given receiptHandles slice.
			for i := range dbe.Info {
				dbe.Info[i].Index = indexMap[dbe.Info[i].Index]
			}

			allInfo = append(allInfo, dbe.Info...)
		}
		batchesDeleted++
	}

	if len(allErrors) > 0 {
		return batchesDeleted, &DeleteNBatchError{
			Errors: allErrors,
			Info:   allInfo,
		}
	}
	return batchesDeleted, nil
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

func (s *SQS) GetQueueARN(url string) (string, error) {

	params := sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(url),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameQueueArn,
		},
	}

	output, err := s.client.GetQueueAttributes(context.TODO(), &params)
	if err != nil {
		return "", err
	}
	arn := output.Attributes[string(types.QueueAttributeNameQueueArn)]
	if arn == "" {
		return "", errors.New("ARN attribute not found")
	}
	return arn, nil
}

// CreateQueue creates an Amazon SQS queue with the specified name. You can specify
// whether the queue is created as a FIFO queue. Returns the queue URL.
func (s *SQS) CreateQueue(queueName string, isFifoQueue bool) (string, error) {

	queueAttributes := map[string]string{}
	if isFifoQueue {
		queueAttributes["FifoQueue"] = "true"
	}
	queue, err := s.client.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName:  aws.String(queueName),
		Attributes: queueAttributes,
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(queue.QueueUrl), err
}

// CheckQueue checks if the given SQS queue exists and is accessible.
func (s *SQS) CheckQueue(queueUrl string) error {
	params := sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(queueUrl),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
	}
	_, err := s.client.GetQueueAttributes(context.TODO(), &params)
	return err
}

// DeleteQueue deletes an Amazon SQS queue.
func (s *SQS) DeleteQueue(queueUrl string) error {
	_, err := s.client.DeleteQueue(context.TODO(), &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueUrl)})

	return err
}

func Cancelled(err error) bool {
	var opErr *smithy.OperationError
	if errors.As(err, &opErr) {
		return opErr.Service() == "SQS" && strings.Contains(opErr.Unwrap().Error(), "context canceled")
	}
	return false
}

func IsNoMessagesError(err error) bool {
	return errors.Is(err, ErrNoMessages)
}
