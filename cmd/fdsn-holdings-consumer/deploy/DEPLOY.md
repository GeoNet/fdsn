# DEPLOY

For deployment to AWS:

## S3 Bucket notifications

* Use an AWS S3 bucket notification to notify about the upload of miniSEED.  This should be create or remove.
* Send the S3 notifications to an SNS topic.  This will need permissions for the S3 bucket to send to it.
* Use SNS->SQS fanout.  The SQS subscription should be for raw messages and SNS needs permissions to send to the SQS queue.
* The role will need permissions to read from the bucket.

## Service on ECS

* Role based permissions are used for access to AWS resources - see `fdsn-holdings-consumer.policy.json`.
* Replace the variables in all caps in `fdsn-holdings-consumer.policy.json` with the deployment values. 
* Create a policy named `fdsn-holdings-consumer` using the file `fdsn-holdings-consumer.policy.json`.
* Create an ECS Task role named role `fdsn-holdings-consumer` that used the `fdsn-holdings-consumer` policy.
* Register an ECS task named `fdsn-holdings-consumer` using the `fdsn-holdings-consumer` role.  
* Deploy the task as a service to an ECS cluster.
