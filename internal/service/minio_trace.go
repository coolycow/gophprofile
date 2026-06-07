package service

import (
	"context"
	"io"

	"github.com/coolycow/gophprofile/internal/observability"
	"github.com/minio/minio-go/v7"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func putObjectTraced(ctx context.Context, client *minio.Client, bucket, key string, r io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "minio.put_object")
	defer span.End()
	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("object_key", key),
		attribute.Int64("file_size", size),
	)
	info, err := client.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return info, err
}

func getObjectTraced(ctx context.Context, client *minio.Client, bucket, key string) (*minio.Object, error) {
	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "minio.get_object")
	defer span.End()
	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("object_key", key),
	)
	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return obj, err
}

func removeObjectTraced(ctx context.Context, client *minio.Client, bucket, key string) error {
	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "minio.remove_object")
	defer span.End()
	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("object_key", key),
	)
	err := client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func statObject(ctx context.Context, client *minio.Client, bucket, key string) (minio.ObjectInfo, error) {
	ctx, span := otel.Tracer(observability.Tracer()).Start(ctx, "minio.stat_object")
	defer span.End()
	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("object_key", key),
	)
	info, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return info, err
}

func (s *avatarService) putObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	return putObjectTraced(ctx, s.minioClient, bucket, key, reader, size, contentType)
}

func (s *avatarService) getObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	return getObjectTraced(ctx, s.minioClient, bucket, key)
}

func (s *workerService) putObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	return putObjectTraced(ctx, s.minioClient, bucket, key, reader, size, contentType)
}

func (s *workerService) getObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	return getObjectTraced(ctx, s.minioClient, bucket, key)
}

func (s *workerService) removeObject(ctx context.Context, bucket, key string) error {
	return removeObjectTraced(ctx, s.minioClient, bucket, key)
}
