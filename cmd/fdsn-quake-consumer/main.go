// fdsn-s3-consumer receives notifications for the creation of SeisComPML objects
// in AWS S3.  Notifications are received from SQS.
// SeisComPML objects are fetched from S3 and stored in the DB.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/aws/sqs"
	"github.com/GeoNet/kit/cfg"
	"github.com/GeoNet/kit/metrics"
)

var (
	queueURL  = os.Getenv("SQS_QUEUE_URL")
	s3Client  s3.S3
	sqsClient sqs.SQS
	db        *sql.DB
)

type notification struct {
	s3.Event
}

func main() {
	p, err := cfg.PostgresEnv()
	if err != nil {
		log.Fatalf("error reading DB config from the environment vars: %s", err)
	}

	db, err = sql.Open("postgres", p.Connection())
	if err != nil {
		log.Fatalf("error with DB config: %s", err)
	}
	defer db.Close()

	db.SetMaxIdleConns(p.MaxIdle)
	db.SetMaxOpenConns(p.MaxOpen)

ping:
	for {
		err = db.Ping()
		if err != nil {
			log.Println("problem pinging DB sleeping and retrying")
			time.Sleep(time.Second * 30)
			continue ping
		}
		break ping
	}

	s3Client, err = s3.NewWithMaxRetries(100)
	if err != nil {
		log.Fatalf("creating S3 client: %s", err)
	}

	sqsClient, err = sqs.NewWithMaxRetries(100)
	if err != nil {
		log.Fatalf("creating SQS client: %s", err)
	}

	log.Println("listening for messages")

	var r sqs.Raw
	var n notification

	for {
		r, err = sqsClient.Receive(queueURL, 600)
		if err != nil {
			log.Printf("problem receiving message, backing off: %s", err)
			time.Sleep(time.Second * 20)
			continue
		}

		err = metrics.DoProcess(&n, []byte(r.Body))
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
func (n *notification) Process(msg []byte) error {
	err := json.Unmarshal(msg, n)
	if err != nil {
		return err
	}

	// add testing on the message.  If these return errors the message should
	// go to the DLQ for further inspectio.  Will catch errors such
	// as SQS->SNS subscriptions being not for raw messages.S
	if n.Records == nil {
		return errors.New("got nil Records pointer in notification message")
	}

	if len(n.Records) == 0 {
		return errors.New("got zero Records in notification message")
	}

	var b bytes.Buffer

	for _, v := range n.Records {
		b.Reset()

		err = s3Client.Get(v.S3.Bucket.Name, v.S3.Object.Key, v.S3.Object.VersionId, &b)
		if err != nil {
			return fmt.Errorf("error getting SC3ML %s %s: %w", v.S3.Bucket.Name, v.S3.Object.Key, err)
		}

		var e event

		if err := unmarshal(b.Bytes(), &e); err != nil {
			return fmt.Errorf("error unmarshalling SC3ML %s %s: %w", v.S3.Bucket.Name, v.S3.Object.Key, err)
		}

		if err := e.save(); err != nil {
			return fmt.Errorf("error saving SC3ML %s %s: %w", v.S3.Bucket.Name, v.S3.Object.Key, err)
		}
	}

	return nil
}
