package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"

	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/logger"
)

const (
	maxImageSize = 5 * 1024 * 1024  // 5MB
	maxVideoSize = 50 * 1024 * 1024 // 50MB

	mediaTypeImage = "image"
	mediaTypeVideo = "video"

	// postMediaFolder は投稿メディアの保存先。オブジェクトパスは posts/{post_id}/{uuid}{拡張子}
	postMediaFolder = "posts"

	// deleteObjectsTimeout はアップロード済みオブジェクトの後始末にかける時間の上限
	deleteObjectsTimeout = 10 * time.Second
)

var (
	ErrFileTooLarge        = errors.New("file too large")
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrStorageUnavailable  = errors.New("storage is not configured")
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

// MediaURLSigner は Storage 上のオブジェクトパスを期限付きの閲覧URLに変換する
type MediaURLSigner interface {
	SignedURL(objectPath string) (string, error)
}

// StorageService はFirebase Storageへのファイルアップロードと閲覧URLの発行を担う
type StorageService interface {
	MediaURLSigner
	// UploadImage はアバター/バナー用の画像を公開オブジェクトとしてアップロードし、公開URLを返す
	UploadImage(ctx context.Context, folder string, fh *multipart.FileHeader) (string, error)
	// UploadPostMedia は投稿用の画像/動画を非公開オブジェクトとしてアップロードし、
	// オブジェクトパスとメディア種別（"image" | "video"）を返す
	UploadPostMedia(ctx context.Context, postID uuid.UUID, fh *multipart.FileHeader) (objectPath string, mediaType string, err error)
	// DeleteObjects はアップロード済みオブジェクトを削除する（DB 保存に失敗したときの後始末）。
	// 失敗してもログに残すだけで呼び出し元の処理は止めない
	DeleteObjects(ctx context.Context, objectPaths []string)
}

// objectStore は Storage バケットへの操作。テストで実バケットに接続せず差し替えるために分けている
type objectStore interface {
	Write(ctx context.Context, objectPath, contentType string, data io.Reader) error
	MakePublic(ctx context.Context, objectPath string) error
	Delete(ctx context.Context, objectPath string) error
	SignedURL(objectPath string, expires time.Time) (string, error)
}

// firebaseObjectStore は Firebase Admin SDK のバケットハンドルを使う objectStore
type firebaseObjectStore struct {
	bucket *storage.BucketHandle
	sa     config.ServiceAccount
}

func (f *firebaseObjectStore) Write(ctx context.Context, objectPath, contentType string, data io.Reader) error {
	// 書き込みの途中で失敗したときに Close せず ctx をキャンセルし、途中までの内容をオブジェクトにしない
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	w := f.bucket.Object(objectPath).NewWriter(ctx)
	w.ContentType = contentType
	if _, err := io.Copy(w, data); err != nil {
		return fmt.Errorf("write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}
	return nil
}

func (f *firebaseObjectStore) MakePublic(ctx context.Context, objectPath string) error {
	if err := f.bucket.Object(objectPath).ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return fmt.Errorf("set acl: %w", err)
	}
	return nil
}

func (f *firebaseObjectStore) Delete(ctx context.Context, objectPath string) error {
	return f.bucket.Object(objectPath).Delete(ctx)
}

// SignedURL は GET 専用の V4 署名付きURLを返す。
// 秘密鍵を明示して渡し、API 呼び出しを伴わないローカル署名にする（SDK の自動検出は IAM signBlob に
// フォールバックすることがあり、一覧表示で画像ごとに通信が発生するのを避ける）
func (f *firebaseObjectStore) SignedURL(objectPath string, expires time.Time) (string, error) {
	return f.bucket.SignedURL(objectPath, &storage.SignedURLOptions{
		GoogleAccessID: f.sa.ClientEmail,
		PrivateKey:     []byte(f.sa.PrivateKey),
		Method:         http.MethodGet,
		Expires:        expires,
		Scheme:         storage.SigningSchemeV4,
	})
}

type storageService struct {
	store        objectStore
	bucketName   string
	signedURLTTL time.Duration
}

// NewStorageService は共有のFirebase Adminアプリからストレージサービスを構築する。
// バケット名・サービスアカウントの不備はリクエスト時ではなく起動時に検出できるよう、ここでエラーを返す
func NewStorageService(cfg *config.Config, app *firebase.App) (StorageService, error) {
	if app == nil {
		return nil, errors.New("firebase app is not initialized")
	}
	if cfg.FirebaseStorageBucket == "" {
		return nil, errors.New("FIREBASE_STORAGE_BUCKET が設定されていません")
	}
	sa, err := config.ParseServiceAccount(cfg.FirebaseCredentials)
	if err != nil {
		return nil, err
	}

	// クライアントはリクエストをまたいで使い回すため、リクエストの ctx ではなく Background で作る
	client, err := app.Storage(context.Background())
	if err != nil {
		return nil, fmt.Errorf("storage client: %w", err)
	}
	bucket, err := client.DefaultBucket()
	if err != nil {
		return nil, fmt.Errorf("default bucket: %w", err)
	}

	store := &firebaseObjectStore{bucket: bucket, sa: sa}
	// 秘密鍵の形式が壊れていると全画像が表示できなくなるので、試しに署名して確認する（通信は発生しない）
	if _, err := store.SignedURL("startup-check", time.Now().Add(time.Minute)); err != nil {
		return nil, errors.New("FIREBASE_CREDENTIALS_JSON の private_key で署名できません")
	}

	return &storageService{store: store, bucketName: cfg.FirebaseStorageBucket, signedURLTTL: cfg.MediaSignedURLTTL}, nil
}

// NewUnavailableStorageService は Firebase の認証情報がない環境（ENV=test）用のサービスを返す。
// アップロード・URL発行は ErrStorageUnavailable になる
func NewUnavailableStorageService() StorageService {
	return &storageService{}
}

func (s *storageService) UploadImage(ctx context.Context, folder string, fh *multipart.FileHeader) (string, error) {
	objectName, _, err := s.upload(ctx, folder, fh, maxImageSize, allowedImageExtensions)
	if err != nil {
		return "", err
	}
	if err := s.store.MakePublic(ctx, objectName); err != nil {
		return "", err
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, objectName), nil
}

func (s *storageService) UploadPostMedia(ctx context.Context, postID uuid.UUID, fh *multipart.FileHeader) (string, string, error) {
	mediaType, maxSize, allowedExt := postMediaRule(detectContentType(fh))
	if mediaType == "" {
		return "", "", ErrUnsupportedFileType
	}
	folder := fmt.Sprintf("%s/%s", postMediaFolder, postID)
	objectName, _, err := s.upload(ctx, folder, fh, maxSize, allowedExt)
	if err != nil {
		return "", "", err
	}
	return objectName, mediaType, nil
}

func (s *storageService) DeleteObjects(ctx context.Context, objectPaths []string) {
	if s.store == nil || len(objectPaths) == 0 {
		return
	}
	// 呼び出し元のリクエストがキャンセル・タイムアウトした後でも後始末できるようにする
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteObjectsTimeout)
	defer cancel()

	for _, objectPath := range objectPaths {
		if err := s.store.Delete(ctx, objectPath); err != nil {
			logger.FromContext(ctx).Warn().Err(err).Str("object_path", objectPath).Msg("failed to delete uploaded object")
		}
	}
}

func (s *storageService) SignedURL(objectPath string) (string, error) {
	if s.store == nil {
		return "", ErrStorageUnavailable
	}
	return s.store.SignedURL(objectPath, time.Now().Add(s.signedURLTTL))
}

// postMediaRule は判定済みのMIMEタイプに対応するメディア種別・サイズ上限・許可する形式を返す。
// 投稿に使えない形式のときは mediaType が空になる
func postMediaRule(contentType string) (mediaType string, maxSize int64, allowedExt map[string]string) {
	switch {
	case allowedImageExtensions[contentType] != "":
		return mediaTypeImage, maxImageSize, allowedImageExtensions
	case allowedVideoExtensions[contentType] != "":
		return mediaTypeVideo, maxVideoSize, allowedVideoExtensions
	default:
		return "", 0, nil
	}
}

// detectContentType はファイルの内容からMIMEタイプを判定する
func detectContentType(fh *multipart.FileHeader) string {
	src, err := fh.Open()
	if err != nil {
		return ""
	}
	defer src.Close()

	mtype, err := mimetype.DetectReader(src)
	if err != nil {
		return ""
	}
	return mtype.String()
}

// upload はファイルを検証してバケットに書き込み、オブジェクトパスとMIMEタイプを返す。
// 拡張子と Content-Type は実際のバイト列から決め、利用者が送ったファイル名やヘッダーは使わない
func (s *storageService) upload(ctx context.Context, folder string, fh *multipart.FileHeader, maxSize int64, allowedExt map[string]string) (string, string, error) {
	if fh.Size > maxSize {
		return "", "", ErrFileTooLarge
	}

	src, err := fh.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	mtype, err := mimetype.DetectReader(src)
	if err != nil {
		return "", "", err
	}
	ext, ok := allowedExt[mtype.String()]
	if !ok {
		return "", "", ErrUnsupportedFileType
	}
	if s.store == nil {
		return "", "", ErrStorageUnavailable
	}

	// 判定で読み進めた分を巻き戻して、全体をメモリに載せずに転送する
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", "", err
	}
	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)
	if err := s.store.Write(ctx, objectName, mtype.String(), src); err != nil {
		return "", "", err
	}
	return objectName, mtype.String(), nil
}
