// fdsn-holdings-consumer receives notifications for the creation of miniSEED objects
// in AWS S3.  Notifications are received from SQS.
// The the miniSEED file referred to by the notification is fetched and indexed.  The
// results are saved to the holdings DB.
//
// Multiple instances (workers) of this code can be run against the same queue for
// Large data reindexing tasks.  Reindexing files that already exist in the bucket
// would require sending messages in the notification format to the SQS queue.
// See github.com/GeoNet/kit/aws/s3 for the Event type.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/aws/sqs"
	"github.com/GeoNet/kit/cfg"
	"github.com/GeoNet/kit/health"
	"github.com/GeoNet/kit/metrics"
)

const (
	healthCheckAged    = 5 * time.Minute  //need to have a good heartbeat within this time
	healthCheckStartup = 5 * time.Minute  //ignore heartbeat messages for this time after starting
	healthCheckTimeout = 30 * time.Second //health check timeout
	healthCheckService = ":7777"          //end point to listen to for SOH checks
	healthCheckPath    = "/soh"
)

var (
	db           *sql.DB
	queueURL     string
	sqsClient    sqs.SQS
	s3Client     s3.S3
	saveHoldings *sql.Stmt
)

type event struct {
	s3.Event
}

// init and check aws variables
func initAwsClient() {
	queueURL = os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL is not set")
	}

	var err error
	sqsClient, err = sqs.NewWithMaxRetries(100)
	if err != nil {
		log.Fatalf("error creating SQS client: %s", err)
	}
	if err = sqsClient.CheckQueue(queueURL); err != nil {
		log.Fatalf("error checking queueURL %s:  %s", queueURL, err.Error())
	}

	s3Client, err = s3.NewWithMaxRetries(3)
	if err != nil {
		log.Fatalf("error creating S3 client: %s", err)
	}
}

func main() {
	//check health
	if health.RunningHealthCheck() {
		healthCheck()
	}

	//run as normal service
	initAwsClient()
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

	// provide a soh heartbeat
	health := health.New(healthCheckService, healthCheckAged, healthCheckStartup)

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

	log.Println("listening for messages")

	var r sqs.Raw
	var e event

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

loop1:
	for {
		health.Ok() // update soh
		r, err = sqsClient.ReceiveWithContext(ctx, queueURL, 600)
		if err != nil {
			switch {
			case sqs.IsNoMessagesError(err):
				continue
			case sqs.Cancelled(err): //stoped
				log.Println("##1 system stop... ")
				break loop1
			default:
				slog.Warn("problem receiving message, backing off", "err", err)
				time.Sleep(time.Second * 20)
			}
			continue
		}

		err = metrics.DoProcess(&e, []byte(r.Body))
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

// check health by calling the http soh endpoint
// cmd: ./fdsn-holdings-consumer  -check
func healthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	msg, err := health.Check(ctx, healthCheckService+healthCheckPath, healthCheckTimeout)
	if err != nil {
		log.Printf("status: %v", err)
		os.Exit(1)
	}
	log.Printf("status: %s", string(msg))
	os.Exit(0)
}

// Process implements msg.Processor for event.
func (e *event) Process(msg []byte) error {
	err := json.Unmarshal(msg, e)
	if err != nil {
		return err
	}

	// add testing on the message.  If these return errors the message should
	// go to the DLQ for further inspectio.  Will catch errors such
	// as SQS->SNS subscriptions being not for raw messages.S
	if e.Records == nil {
		return errors.New("got nil Records pointer in notification message")
	}

	if len(e.Records) == 0 {
		return errors.New("got zero Records in notification message")
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
				return fmt.Errorf("error saving holding for %s %s: %w", v.S3.Bucket.Name, v.S3.Object.Key, err)
			}

		case strings.HasPrefix(v.EventName, "ObjectRemoved"):
			h := holding{key: v.S3.Object.Key}
			err = h.delete()
			if err != nil {
				return fmt.Errorf("error deleting holdings for %s %s: %w", v.S3.Bucket.Name, v.S3.Object.Key, err)
			}
		default:
			return errors.New("unknown EventName: " + v.EventName)
		}
	}

	return nil
}
