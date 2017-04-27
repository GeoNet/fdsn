// fdsn-holdings-consumer receives notifications for the creation of miniSEED objects
// in AWS S3.  Notifications are received from SQS.
// The key (string) for miniSEED objects are posted to the FDSN webservices for indexing in the data holdings.
package main

import (
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
	"strings"
	"time"
)

var (
	key       = os.Getenv("FDSN_KEY")
	path      = os.Getenv("FDSN_HOLDINGS_URL")
	queueURL  = os.Getenv("SQS_QUEUE_URL")
	client    *http.Client
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

	sqsClient, err = sqs.New(100)
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

	for _, v := range e.Records {
		switch {
		case strings.HasPrefix(v.EventName, "ObjectCreated"):
			err = fdsn("PUT", v.S3.Object.Key)
		case strings.HasPrefix(v.EventName, "ObjectRemoved"):
			err = fdsn("DELETE", v.S3.Object.Key)
		default:
			err = errors.New("unknown EventName: " + v.EventName)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// fdsn puts the objectKey to the FDSN holdings webservice.
func fdsn(method, objectKey string) error {
	req, err := http.NewRequest(method, path+"/"+objectKey, nil)
	if err != nil {
		return err
	}

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
