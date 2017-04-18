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
	EventName string
	S3        EventS3
}

type EventS3 struct {
	Object EventObject
	Bucket EventBucket
}

type EventBucket struct {
	Name string
}

type EventObject struct {
	Key       string
	VersionId string
}
