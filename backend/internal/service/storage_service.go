package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"

	"github.com/atyahara/sns-backend/internal/config"
)

const (
	maxImageSize = 5 * 1024 * 1024   // 5MB
	maxVideoSize = 100 * 1024 * 1024 // 100MB
)

var (
	ErrFileTooLarge        = errors.New("file too large")
	ErrUnsupportedFileType = errors.New("unsupported file type")
)

var allowedImageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var allowedVideoExtensions = map[string]string{
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
}

// StorageService はFirebase Storageへのファイルアップロードを担う
type StorageService interface {
	UploadImage(ctx context.Context, folder string, fh *multipart.FileHeader) (string, error)
	// UploadPostMedia は投稿用の画像/動画をアップロードし、公開URLとメディア種別（"image" | "video"）を返す
	UploadPostMedia(ctx context.Context, folder string, fh *multipart.FileHeader) (url string, mediaType string, err error)
}

type storageService struct {
	app        *firebase.App
	bucketName string
}

// NewStorageService は共有のFirebase Adminアプリからストレージサービスを構築する
func NewStorageService(cfg *config.Config, app *firebase.App) StorageService {
	return &storageService{app: app, bucketName: cfg.FirebaseStorageBucket}
}

func (s *storageService) UploadImage(ctx context.Context, folder string, fh *multipart.FileHeader) (string, error) {
	url, _, err := s.upload(ctx, folder, fh, maxImageSize, allowedImageExtensions)
	return url, err
}

func (s *storageService) UploadPostMedia(ctx context.Context, folder string, fh *multipart.FileHeader) (string, string, error) {
	contentType := detectContentType(fh)
	switch {
	case allowedImageExtensions[contentType] != "":
		url, _, err := s.upload(ctx, folder, fh, maxImageSize, allowedImageExtensions)
		return url, "image", err
	case allowedVideoExtensions[contentType] != "":
		url, _, err := s.upload(ctx, folder, fh, maxVideoSize, allowedVideoExtensions)
		return url, "video", err
	default:
		return "", "", ErrUnsupportedFileType
	}
}

// detectContentType はファイルの内容からMIMEタイプを判定する
func detectContentType(fh *multipart.FileHeader) string {
	src, err := fh.Open()
	if err != nil {
		return ""
	}
	defer src.Close()

	head := make([]byte, 3072)
	n, _ := src.Read(head)
	return mimetype.Detect(head[:n]).String()
}

func (s *storageService) upload(ctx context.Context, folder string, fh *multipart.FileHeader, maxSize int64, allowedExt map[string]string) (string, string, error) {
	if fh.Size > maxSize {
		return "", "", ErrFileTooLarge
	}

	src, err := fh.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return "", "", err
	}

	mtype := mimetype.Detect(data)
	ext, ok := allowedExt[mtype.String()]
	if !ok {
		return "", "", ErrUnsupportedFileType
	}

	client, err := s.app.Storage(ctx)
	if err != nil {
		return "", "", fmt.Errorf("storage client: %w", err)
	}
	bucket, err := client.DefaultBucket()
	if err != nil {
		return "", "", fmt.Errorf("default bucket: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)
	obj := bucket.Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = mtype.String()
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		w.Close()
		return "", "", fmt.Errorf("write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", "", fmt.Errorf("close writer: %w", err)
	}

	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", "", fmt.Errorf("set acl: %w", err)
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, objectName), mtype.String(), nil
}
