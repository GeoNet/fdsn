package s3_test

import (
	"encoding/json"
	. "github.com/GeoNet/fdsn/internal/kit/s3"
	"testing"
)

// s3 notification message structure from http://docs.aws.amazon.com/AmazonS3/latest/dev/notification-content-structure.html
var testNotificationMessage = `
{
   "Records":[
      {
         "eventVersion":"2.0",
         "eventSource":"aws:s3",
         "awsRegion":"us-east-1",
         "eventTime":"1970-01-01T00:00:00.000Z",
         "eventName":"ObjectCreated:Put",
         "userIdentity":{
            "principalId":"AIDAJDPLRKLG7UEXAMPLE"
         },
         "requestParameters":{
            "sourceIPAddress":"127.0.0.1"
         },
         "responseElements":{
            "x-amz-request-id":"C3D13FE58DE4C810",
            "x-amz-id-2":"FMyUVURIY8/IgAtTv8xRjskZQpcIZ9KG4V5Wp6S7S/JRWeUWerMUE5JgHvANOjpD"
         },
         "s3":{
            "s3SchemaVersion":"1.0",
            "configurationId":"testConfigRule",
            "bucket":{
               "name":"mybucket",
               "ownerIdentity":{
                  "principalId":"A3NL1KOZZKExample"
               },
               "arn":"arn:aws:s3:::mybucket"
            },
            "object":{
               "key":"HappyFace.jpg",
               "size":1024,
               "eTag":"d41d8cd98f00b204e9800998ecf8427e",
               "versionId":"096fKKXTRTtl3on89fVO.nfljtsv6qko",
               "sequencer":"0055AED6DCD90281E5"
            }
         }
      }
   ]
}
`

func TestNotificationParse(t *testing.T) {
	var e Event

	err := json.Unmarshal([]byte(testNotificationMessage), &e)
	if err != nil {
		t.Error(err)
	}

	if len(e.Records) != 1 {
		t.Error("expected 1 record.")
	}

	r := e.Records[0]

	if r.EventName != "ObjectCreated:Put" {
		t.Errorf("expected event name ObjectCreated:Put got %s", r.EventName)
	}

	if r.S3.Bucket.Name != "mybucket" {
		t.Errorf("expected mybucket got %s", r.S3.Bucket.Name)
	}

	if r.S3.Object.Key != "HappyFace.jpg" {
		t.Errorf("expected HappyFace.jpg got %s", r.S3.Object.Key)
	}

	if r.S3.Object.VersionId != "096fKKXTRTtl3on89fVO.nfljtsv6qko" {
		t.Errorf("expected versionID 096fKKXTRTtl3on89fVO.nfljtsv6qko got %s", r.S3.Object.VersionId)
	}
}
