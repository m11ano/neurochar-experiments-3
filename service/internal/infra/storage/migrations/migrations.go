package migrations

import (
	"context"
	"errors"

	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
)

type bucketListItem struct {
	name   storage.BucketName
	policy string
}

var bucketsList = []bucketListItem{
	{
		name:   storage.BucketCommonFiles,
		policy: bucketPolicy(storage.BucketCommonFiles),
	},
}

func UpBuckets(ctx context.Context, client storage.Client) (bool, error) {
	createdAny := false

	for _, item := range bucketsList {
		err := client.CreateBucket(ctx, item.name, item.policy)
		if err != nil && !errors.Is(err, storage.ErrBucketAlreadyExists) {
			return false, err
		}

		if err == nil {
			createdAny = true
		}
	}

	return createdAny, nil
}
