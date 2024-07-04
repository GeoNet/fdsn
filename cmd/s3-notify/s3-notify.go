package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/aws/sqs"
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
}

func initAWS() {
	var err error
	s3c, err := s3.NewWithMaxRetries(100)
	if err != nil {
		log.Fatalf("error creating S3 client: %s", err)
	}
	s3Client = &s3c
	sqsClient, err = sqs.NewWithMaxRetries(100)
	if err != nil {
		log.Fatalf("error creating SQS client: %s", err)
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

	initAWS()
	keys, err := s3Client.ListAll(bucketName, keyPrefix)
	if err != nil {
		log.Fatalf("error listing S3 objects: %s", err)
	}
	for _, k := range keys {
		if strings.HasSuffix(k, "/") {
			continue // directories have trailing /
		}
		if err := sendSQS(k); err != nil {
			log.Fatal(err)
			break
		}
	}

	fmt.Println("Total keys matched:", len(keys))
}

func sendSQS(key string) error {
	fmt.Println("Key:", key)

	if sqsUrl == "" {
		return nil
	}

	e := s3.Event{
		Records: []s3.EventRecord{
			{
				EventName: "ObjectCreated:Put",
				S3: s3.EventS3{
					Object: s3.EventObject{
						Key: key,
					},
					Bucket: s3.EventBucket{
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

	return sqsClient.Send(sqsUrl, string(b))
}
