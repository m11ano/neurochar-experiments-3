package migrations

import (
	"fmt"

	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
)

func bucketPolicy(bucket storage.BucketName) string {
	return fmt.Sprintf(`{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Principal": {
				"AWS": [
					"*"
				]
			},
			"Action": [
				"s3:GetObject"
			],
			"Resource": [
				"arn:aws:s3:::%s/*"
			]
		}
	]
}`, bucket)
}
