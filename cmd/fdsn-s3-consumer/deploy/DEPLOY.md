# DEPLOY

For deployment to AWS:

## S3 Bucket notifications

* Use an AWS S3 bucket notification to notify about the upload of SC3ML.
* Use SNS->SQS fanout.  The SQS subscription should be for raw messages and SNS needs permissions to send to the SQS queue.

## Service on ECS

* Role based permissions are used for access to AWS resources - see `fdsn-s3-consumer.policy.json`.
* Replace the variables in all caps in `fdsn-s3-consumer.policy.json` with the deployment values. 
* Create a policy named `fdsn-s3-consumer` using the file `fdsn-s3-consumer.policy.json`.
* Create an ECS Task role named role `fdsn-s3-consumer` that used the `fdsn-s3-consumer` policy.
* Register an ECS task named `fdsn-s3-consumer` using the `fdsn-s3-consumer` role.  For a guide see `fdsn-s3-consumer.task.json`
Note at the least the values `SET_VALUE` need replacing with deployment values.
* Deploy the task as a service to an ECS cluster.
