// fdsn-holdings-consumer receives notifications for the creation of miniSEED objects
// in AWS S3.  Notifications are received from SQS.
// The the miniSEED file referred to by the notification is fetched and indexed.  The
// results are saved to the holdings DB.
//
// Multiple instances (workers) of this code can be run against the same queue for
// Large data reindexing tasks.  Reindexing files that already exist in the bucket
// would require sending messages in the notification format to the SQS queue.
// See github.com/GeoNet/fdsn/internal/platform/s3 for the Event type.
package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/GeoNet/fdsn/internal/ddogmsg"
	"github.com/GeoNet/fdsn/internal/platform/cfg"
	"github.com/GeoNet/fdsn/internal/platform/msg"
	nf "github.com/GeoNet/fdsn/internal/platform/s3"
	"github.com/GeoNet/fdsn/internal/platform/sqs"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"log"
	"os"
	"strings"
	"time"
)

var (
	db           *sql.DB
	queueURL     = os.Getenv("SQS_QUEUE_URL")
	sqsClient    sqs.SQS
	s3Session    *session.Session
	s3Client     *s3.S3
	saveHoldings *sql.Stmt
)

type event struct {
	nf.Event
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

	// TODO - this is duplicated in the test set up.
	// make a struct like in fdsn-holdings-consumer and move the
	// db connection and set up to that.
	//
	// when a miniSEED file has errors the error state and message are saved to
	// the holdings db.  The key is the file name and the streamPK will be
	// based on a nscl with zero strings "".""."".""
	// if the error is corrected the stream will change to some valid nscl.
	// To handle this the streamPK is updated on conflict.
	saveHoldings, err = db.Prepare(`INSERT INTO fdsn.holdings (streamPK, start_time, numsamples, key, error_data, error_msg)
	SELECT streamPK, $5, $6, $7, $8, $9
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4
	ON CONFLICT (streamPK, key) DO UPDATE SET
	streamPK = EXCLUDED.streamPK,
	start_time = EXCLUDED.start_time,
	numsamples = EXCLUDED.numsamples,
	error_data = EXCLUDED.error_data,
	error_msg = EXCLUDED.error_msg`)
	if err != nil {
		log.Fatalf("preparing saveHoldings statement: %s", err.Error())
	}

	defer saveHoldings.Close()

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

	sqsClient, err = sqs.New(100)
	if err != nil {
		log.Fatalf("creating SQS client: %s", err)
	}

	s3Session, err = session.NewSession()
	if err != nil {
		log.Fatalf("creating S3 session: %s", err)
	}

	s3Session.Config.Retryer = client.DefaultRetryer{NumMaxRetries: 3}
	s3Client = s3.New(s3Session)

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
		return nil
	}

	for _, v := range e.Records {
		switch {
		case strings.HasPrefix(v.EventName, "ObjectCreated"):
			// TODO (GMC) setting errors like this will include miniSEED errors as well
			// errors from reading from S3.  Is this ok or should it just be miniSEED errors?
			h, err := holdingS3(v.S3.Bucket.Name, v.S3.Object.Key)
			if err != nil {
				h.key = v.S3.Object.Key
				h.errorData = true
				h.errorMsg = err.Error()
			}

			err = h.save()
			if err != nil {
				return errors.Wrapf(err, "error saving holding for % %", v.S3.Bucket.Name, v.S3.Object.Key)
			}

		case strings.HasPrefix(v.EventName, "ObjectRemoved"):
			h := holding{key: v.S3.Object.Key}
			err = h.delete()
			if err != nil {
				return errors.Wrapf(err, "error deleting holdings for %s %s", v.S3.Bucket.Name, v.S3.Object.Key)
			}
		default:
			return errors.New("unknown EventName: " + v.EventName)
		}
	}

	return nil
}
