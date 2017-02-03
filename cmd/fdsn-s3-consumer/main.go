// fdsn-s3-consumer receives notifications for the creation of SeisComPML objects
// in AWS S3.  Notifications are received from SQS.
// SeisComPML objects are fetched from S3 and posted to the FDSN webservices.
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GeoNet/fdsn/internal/kit/msg"
	"github.com/GeoNet/fdsn/internal/kit/s3"
	"github.com/GeoNet/fdsn/internal/kit/sqs"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	key       = os.Getenv("FDSN_KEY")
	path      = os.Getenv("FDSN_SC3ML_URL")
	queueURL  = os.Getenv("SQS_QUEUE_URL")
	client    *http.Client
	s3Client  s3.S3
	sqsClient sqs.SQS
)

type event struct {
	s3.Event
}

func main() {
	// TODO - remove skip veryifying the TLS cert - currently using a geonet domain cert without a geonet DNS entry.
	client = &http.Client{
		Timeout: time.Duration(60 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var err error

	s3Client, err = s3.New()
	if err != nil {
		log.Fatalf("creating S3 client: %s", err)
	}

	sqsClient, err = sqs.New()
	if err != nil {
		log.Fatalf("creating SQS client: %s", err)
	}

	log.Println("listening for messages")

	var r sqs.Raw
	var e event

	for {
		r, err = sqsClient.Receive(queueURL, 600)
		if err != nil {
			log.Printf("problem receiving message, backing off: %s", err)
			time.Sleep(time.Second * 20)
			continue
		}

		err = msg.DoProcess(&e, []byte(r.Body))
		if err != nil {
			log.Printf("problem processing message, skipping deletion for redelivery: %s", err)
			continue
		}

		err = sqsClient.Delete(queueURL, r.ReceiptHandle)
		if err != nil {
			log.Printf("problem deleting message, continuing: %s", err)
		}
	}
}

// Process implements msg.Processor for event.
func (e *event) Process(msg []byte) error {
	err := json.Unmarshal(msg, e)
	if err != nil {
		return err
	}

	if e.Records == nil || len(e.Records) == 0 {
		return errors.New("received message with no content.")
	}

	var b bytes.Buffer

	for _, v := range e.Records {
		b.Reset()

		err = s3Client.Get(v.S3.Bucket.Name, v.S3.Object.Key, v.S3.Object.VersionId, &b)
		if err != nil {
			return err
		}

		err = fdsn(&b)
		if err != nil {
			return err
		}
	}

	return nil
}

// fdsn posts the SC3ML in b to the FDSN webservice.
func fdsn(b *bytes.Buffer) error {
	req, err := http.NewRequest("POST", path, b)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	req.SetBasicAuth("", key)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("non 200 response (%d)", res.StatusCode)
}
