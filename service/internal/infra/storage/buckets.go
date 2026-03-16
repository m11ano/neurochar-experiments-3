package storage

import (
	"strings"

	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
)

type BucketName string

const (
	BucketCommonFiles BucketName = "neurochar-experiments-1-files"
)

func GetBucketURL(bucket BucketName, cfg *config.Config) string {
	var builder strings.Builder
	builder.WriteString(cfg.Storage.S3URL)
	builder.WriteString("/")
	builder.WriteString(string(bucket))

	return builder.String()
}
