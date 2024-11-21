// fdsn-s3-consumer receives notifications for the creation of SeisComPML objects
// in AWS S3.  Notifications are received from SQS.
// SeisComPML objects are fetched from S3 and stored in the DB.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/aws/sqs"
	"github.com/GeoNet/kit/cfg"
	"github.com/GeoNet/kit/health"
	"github.com/GeoNet/kit/metrics"
	"github.com/GeoNet/kit/slogger"
)

const (
	healthCheckAged    = 5 * time.Minute  //need to have a good heartbeat within this time (depends on tilde-bundle)
	healthCheckStartup = 5 * time.Minute  //ignore heartbeat messages for this time after starting
	healthCheckTimeout = 30 * time.Second //health check timeout
	healthCheckService = ":7777"          //end point to listen to for SOH checks
	healthCheckPath    = "/soh"
)

var (
	queueURL  = os.Getenv("SQS_QUEUE_URL")
	s3Client  s3.S3
	sqsClient sqs.SQS
	db        *sql.DB

	sLogger = slogger.NewSmartLogger(10*time.Minute, "") // log repeated error messages
)

type notification struct {
	s3.Event
}

func main() {
	//check health
	if health.RunningHealthCheck() {
		healthCheck()
	}

	//run as normal service
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

	// provide a soh heartbeat
	health := health.New(healthCheckService, healthCheckAged, healthCheckStartup)

ping:
	for {
		err = db.Ping()
		if err != nil {
			log.Println("problem pinging DB sleeping and retrying")
			health.Ok() //send heartbeat

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

loop1:
	for {
		r, err = sqsClient.ReceiveWithContext(ctx, queueURL, 600)
		if err != nil {
			switch {
			case sqs.Cancelled(err): //stoped
				log.Println("##1 system stop... ")
				break loop1
			case sqs.IsNoMessagesError(err):
				n := sLogger.Log(err)
				if n%100 == 0 { //don't log all repeated error messages
					log.Printf("no message received for %d times ", n)
				}
			default:
				slog.Warn("problem receiving message, backing off", "err", err)
				time.Sleep(time.Second * 20)
			}
			// update soh
			health.Ok()
			continue
		}

		err = metrics.DoProcess(&n, []byte(r.Body))
		if err != nil {
			log.Printf("problem processing message, skipping deletion for redelivery: %s", err)
			// update soh
			health.Ok()
			continue
		}

		err = sqsClient.Delete(queueURL, r.ReceiptHandle)
		if err != nil {
			log.Printf("problem deleting message, continuing: %s", err)
		}
		// update soh
		health.Ok()
	}
}

// check health by calling the http soh endpoint
// cmd: ./fdsn-quake-consumer  -check
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

		log.Println("saved", e.PublicID)
	}

	return nil
}
