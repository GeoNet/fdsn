// fdsn-s3-consumer receives notifications for the creation of SeisComPML objects
// in AWS S3.  Notifications are received from SQS.
// SeisComPML objects are fetched from S3 and stored in the DB.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/GeoNet/fdsn/internal/kit/msg"
	"github.com/GeoNet/fdsn/internal/kit/s3"
	"github.com/GeoNet/fdsn/internal/kit/sqs"
	"github.com/pkg/errors"
	"log"
	"os"
	"time"
	"github.com/GeoNet/fdsn/internal/kit/cfg"
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

	s3Client, err = s3.New(100)
	if err != nil {
		log.Fatalf("creating S3 client: %s", err)
	}

	sqsClient, err = sqs.New(100)
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

		err = msg.DoProcess(&n, []byte(r.Body))
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

	if n.Records == nil || len(n.Records) == 0 {
		return nil
	}

	var b bytes.Buffer

	for _, v := range n.Records {
		b.Reset()

		err = s3Client.Get(v.S3.Bucket.Name, v.S3.Object.Key, v.S3.Object.VersionId, &b)
		if err != nil {
			return errors.Wrapf(err, "error getting SC3ML %s %s", v.S3.Bucket.Name, v.S3.Object.Key)
		}

		var e event

		if err := unmarshal(b.Bytes(), &e); err != nil {
			return errors.Wrapf(err, "error unmarshalling SC3ML %s %s", v.S3.Bucket.Name, v.S3.Object.Key)
		}

		if err := e.save(); err != nil {
			return errors.Wrapf(err, "error saving SC3ML %s %s", v.S3.Bucket.Name, v.S3.Object.Key)
		}
	}

	return nil
}
