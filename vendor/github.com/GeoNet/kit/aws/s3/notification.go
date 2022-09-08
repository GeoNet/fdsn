package s3

// Event is for unmarshaling AWS S3 event notifications.
// http://docs.aws.amazon.com/AmazonS3/latest/dev/notification-content-structure.html
// eventName values from creating and deleting objects via web UI:
// ObjectRemoved:Delete
// ObjectCreated:Put

type Event struct {
	Records []EventRecord
}

type EventRecord struct {
	EventName string  `json:"eventName,omitempty"`
	S3        EventS3 `json:"s3,omitempty"`
}

type EventS3 struct {
	Object EventObject `json:"object,omitempty"`
	Bucket EventBucket `json:"bucket,omitempty"`
}

type EventBucket struct {
	Name string `json:"name,omitempty"`
}

type EventObject struct {
	Key       string `json:"key,omitempty"`
	VersionId string `json:"versionId,omitempty"`
}
