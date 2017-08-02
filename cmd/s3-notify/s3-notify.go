package main

import (
	"encoding/json"
	"flag"
	"fmt"
	fdsnS3 "github.com/GeoNet/fdsn/internal/platform/s3"
	"github.com/GeoNet/fdsn/internal/platform/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
)

var (
	bucketName string
	keyPrefix  string
	sqsUrl     string
	s3Client   *s3.S3
	sqsClient  sqs.SQS
)

func init() {
	flag.StringVar(&bucketName, "bucket-name", "", "S3 bucket name which holds miniSEEDs.")
	flag.StringVar(&keyPrefix, "key-prefix", "", "Key prefix to search in the S3 bucket.")
	flag.StringVar(&sqsUrl, "sqs-url", "", "SQS queue url to send notifications to. Omit this parameter to show the list of matched keys only.")
	flag.Parse()

	s3Session, err := session.NewSession()
	if err != nil {
		log.Fatalf("creating S3 session: %s", err)
	}

	s3Client = s3.New(s3Session)
	sqsClient, err = sqs.New(100)
	if err != nil {
		log.Fatalf("creating sqs: %s", err)
	}
}

func main() {
	if bucketName == "" {
		flag.Usage()
		return
	}
	fmt.Println("Checking S3 bucket:", bucketName)
	fmt.Println("Search key prefix:", keyPrefix)
	if sqsUrl == "" {
		fmt.Println("No sqs-url specified. Displaying matched key only.")
	} else {
		fmt.Println("Send to SQS:", sqsUrl)
	}

	params := s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(keyPrefix),
	}

	cnt := 0
	err := s3Client.ListObjectsV2Pages(&params,
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, o := range page.Contents {
				if *o.Size > 0 { // directories has the size of 0
					if err := sendSQS(o); err != nil {
						log.Fatal(err)
						break
					}
					cnt++
				}
			}
			return !lastPage
		})

	if err != nil {
		log.Fatalf("listing s3 objects:", err)
	}

	fmt.Println("Total keys matched:", cnt)
}

func sendSQS(o *s3.Object) error {
	fmt.Println("Key:", *o.Key)

	if sqsUrl == "" {
		return nil
	}

	e := fdsnS3.Event{
		Records: []fdsnS3.EventRecord{
			{
				EventName: "ObjectCreated:Put",
				S3: fdsnS3.EventS3{
					Object: fdsnS3.EventObject{
						Key: *o.Key,
					},
					Bucket: fdsnS3.EventBucket{
						Name: bucketName,
					},
				},
			},
		},
	}

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return sqsClient.Send(sqsUrl, sqs.Raw{Body: string(b)})
}
