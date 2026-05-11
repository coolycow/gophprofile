package minio

import (
	"context"

	"github.com/coolycow/gophprofile/internal/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func NewMinioClient(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*minio.Client, error) {
	// Инициализируем MinIO клиент
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Log.Fatal("Failed to initialize minio client", zap.Error(err))
	}

	logger.Log.Info("Initialized minio client successfully")

	// Создаем бакет если он не существует
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			logger.Log.Info("We already own %s", zap.String("bucket", bucketName))
		} else {
			logger.Log.Fatal("Failed to create bucket", zap.Error(err))
		}
	}

	logger.Log.Info("Created bucket successfully")

	return minioClient, nil
}
