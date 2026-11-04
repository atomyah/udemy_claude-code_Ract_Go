package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"regexp"
	"strconv"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"

	"github.com/atyahara/sns-backend/internal/config"
)

// fakeObjectStore は実バケットに接続せず、Storage への操作を記録する objectStore
type fakeObjectStore struct {
	written      map[string][]byte
	contentTypes map[string]string
	public       []string
	deleted      []string
	writeErr     error
	deleteErr    error
	signedExpiry time.Time
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{written: map[string][]byte{}, contentTypes: map[string]string{}}
}

func (f *fakeObjectStore) Write(_ context.Context, objectPath, contentType string, data io.Reader) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	content, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	f.written[objectPath] = content
	f.contentTypes[objectPath] = contentType
	return nil
}

func (f *fakeObjectStore) MakePublic(_ context.Context, objectPath string) error {
	f.public = append(f.public, objectPath)
	return nil
}

func (f *fakeObjectStore) Delete(_ context.Context, objectPath string) error {
	f.deleted = append(f.deleted, objectPath)
	return f.deleteErr
}

func (f *fakeObjectStore) SignedURL(objectPath string, expires time.Time) (string, error) {
	f.signedExpiry = expires
	return "https://signed.example.com/" + objectPath, nil
}

// TestStorageService_Upload_SizeLimits は画像5MB・動画50MBの上限が
// Firebaseへアップロードする前に効いていることを検証する。
// サイズ判定はファイルを開く前に行われるため、実体のないFileHeaderで検証できる。
func TestStorageService_Upload_SizeLimits(t *testing.T) {
	tests := []struct {
		name         string
		size         int64
		maxSize      int64
		wantTooLarge bool
	}{
		{name: "画像5MBちょうどは上限で弾かれない", size: maxImageSize, maxSize: maxImageSize, wantTooLarge: false},
		{name: "画像5MB+1バイトは弾かれる", size: maxImageSize + 1, maxSize: maxImageSize, wantTooLarge: true},
		{name: "動画50MBちょうどは上限で弾かれない", size: maxVideoSize, maxSize: maxVideoSize, wantTooLarge: false},
		{name: "動画50MB+1バイトは弾かれる", size: maxVideoSize + 1, maxSize: maxVideoSize, wantTooLarge: true},
	}

	svc := &storageService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := &multipart.FileHeader{Filename: "test.bin", Size: tt.size}

			_, _, err := svc.upload(context.Background(), "posts", fh, tt.maxSize, allowedImageExtensions)

			if tt.wantTooLarge {
				assert.ErrorIs(t, err, ErrFileTooLarge)
			} else {
				// 上限以内なのでサイズでは弾かれない（この先のファイル読み込みで別のエラーになる）
				assert.NotErrorIs(t, err, ErrFileTooLarge)
			}
		})
	}
}

// TestStorageService_UploadImage_UsesImageSizeLimit は
// アバター/バナー用のUploadImageが画像の上限（5MB）を使っていることを検証する
func TestStorageService_UploadImage_UsesImageSizeLimit(t *testing.T) {
	svc := &storageService{}

	_, err := svc.UploadImage(context.Background(), "avatars",
		&multipart.FileHeader{Filename: "avatar.png", Size: maxImageSize + 1})

	assert.ErrorIs(t, err, ErrFileTooLarge)
}

// TestStorageService_SizeLimitConstants は仕様（画像5MB・動画50MB）どおりの
// 定数が設定されていることを検証する
func TestStorageService_SizeLimitConstants(t *testing.T) {
	assert.Equal(t, int64(5*1024*1024), int64(maxImageSize))
	assert.Equal(t, int64(50*1024*1024), int64(maxVideoSize))
}

// jpegBytes / webpBytes はmimetypeが各形式と判定する最小限のバイト列
var (
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	webpBytes = []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
)

func TestStorageService_UploadPostMedia_SavesPrivateObjectUnderPostFolder(t *testing.T) {
	tests := []struct {
		name            string
		content         []byte
		wantExt         string
		wantContentType string
		wantMediaType   string
	}{
		{name: "PNG", content: pngBytes, wantExt: ".png", wantContentType: "image/png", wantMediaType: "image"},
		{name: "JPEG", content: jpegBytes, wantExt: ".jpg", wantContentType: "image/jpeg", wantMediaType: "image"},
		{name: "WebP", content: webpBytes, wantExt: ".webp", wantContentType: "image/webp", wantMediaType: "image"},
		{name: "MP4", content: mp4Bytes, wantExt: ".mp4", wantContentType: "video/mp4", wantMediaType: "video"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeObjectStore()
			svc := &storageService{store: store}
			postID := uuid.New()

			objectPath, mediaType, err := svc.UploadPostMedia(context.Background(), postID, newFileHeaders(t, tt.content)[0])

			require.NoError(t, err)
			assert.Equal(t, tt.wantMediaType, mediaType)
			assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`^posts/%s/[0-9a-f-]{36}%s$`, postID, regexp.QuoteMeta(tt.wantExt))), objectPath)
			assert.Equal(t, tt.content, store.written[objectPath])
			assert.Equal(t, tt.wantContentType, store.contentTypes[objectPath])
			// 署名付きURLで配信するため公開設定はしない
			assert.Empty(t, store.public)
		})
	}
}

func TestStorageService_UploadPostMedia_IgnoresClientFilename(t *testing.T) {
	store := newFakeObjectStore()
	svc := &storageService{store: store}
	fh := newFileHeaders(t, pngBytes)[0]
	fh.Filename = "../../evil.html"

	objectPath, _, err := svc.UploadPostMedia(context.Background(), uuid.New(), fh)

	require.NoError(t, err)
	assert.NotContains(t, objectPath, "evil")
	assert.NotContains(t, objectPath, "..")
}

func TestStorageService_UploadPostMedia_RejectsDisguisedFile(t *testing.T) {
	store := newFakeObjectStore()
	svc := &storageService{store: store}
	// 拡張子は画像でも中身がテキストなら受け付けない
	fh := newFileHeaders(t, []byte("<script>alert(1)</script>"))[0]
	fh.Filename = "photo.png"

	_, _, err := svc.UploadPostMedia(context.Background(), uuid.New(), fh)

	assert.ErrorIs(t, err, ErrUnsupportedFileType)
	assert.Empty(t, store.written)
}

func TestStorageService_UploadPostMedia_ImageTooLarge(t *testing.T) {
	store := newFakeObjectStore()
	svc := &storageService{store: store}
	fh := newFileHeaders(t, pngBytes)[0]
	fh.Size = maxImageSize + 1

	_, _, err := svc.UploadPostMedia(context.Background(), uuid.New(), fh)

	assert.ErrorIs(t, err, ErrFileTooLarge)
	assert.Empty(t, store.written)
}

func TestStorageService_UploadPostMedia_WriteError(t *testing.T) {
	store := newFakeObjectStore()
	store.writeErr = assert.AnError
	svc := &storageService{store: store}

	_, _, err := svc.UploadPostMedia(context.Background(), uuid.New(), newFileHeaders(t, pngBytes)[0])

	assert.ErrorIs(t, err, assert.AnError)
}

func TestStorageService_UploadImage_MakesObjectPublic(t *testing.T) {
	store := newFakeObjectStore()
	svc := &storageService{store: store, bucketName: "test-bucket"}

	publicURL, err := svc.UploadImage(context.Background(), "avatars", newFileHeaders(t, pngBytes)[0])

	require.NoError(t, err)
	require.Len(t, store.public, 1)
	assert.Equal(t, "https://storage.googleapis.com/test-bucket/"+store.public[0], publicURL)
}

func TestStorageService_DeleteObjects_ContinuesAfterFailure(t *testing.T) {
	store := newFakeObjectStore()
	store.deleteErr = assert.AnError
	svc := &storageService{store: store}

	svc.DeleteObjects(context.Background(), []string{"posts/a.png", "posts/b.png"})

	assert.Equal(t, []string{"posts/a.png", "posts/b.png"}, store.deleted)
}

func TestStorageService_SignedURL_UsesConfiguredTTL(t *testing.T) {
	store := newFakeObjectStore()
	svc := &storageService{store: store, signedURLTTL: 30 * time.Minute}

	signed, err := svc.SignedURL("posts/a.png")

	require.NoError(t, err)
	assert.Equal(t, "https://signed.example.com/posts/a.png", signed)
	assert.WithinDuration(t, time.Now().Add(30*time.Minute), store.signedExpiry, time.Minute)
}

func TestUnavailableStorageService_ReturnsError(t *testing.T) {
	svc := NewUnavailableStorageService()

	_, err := svc.SignedURL("posts/a.png")
	assert.ErrorIs(t, err, ErrStorageUnavailable)

	_, _, err = svc.UploadPostMedia(context.Background(), uuid.New(), newFileHeaders(t, pngBytes)[0])
	assert.ErrorIs(t, err, ErrStorageUnavailable)

	_, err = svc.UploadImage(context.Background(), "avatars", newFileHeaders(t, pngBytes)[0])
	assert.ErrorIs(t, err, ErrStorageUnavailable)

	// 何も起きないこと（panic しないこと）
	svc.DeleteObjects(context.Background(), []string{"posts/a.png"})
}

func TestNewStorageService_RejectsInvalidConfig(t *testing.T) {
	_, err := NewStorageService(&config.Config{FirebaseStorageBucket: "b", FirebaseCredentials: "{}"}, nil)
	assert.Error(t, err, "Firebase アプリがないときはエラー")
}

// newTestPrivateKey はテスト用の使い捨てRSA鍵をPEM形式で生成する（本物の鍵は使わない）
func newTestPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, pem.Encode(&buf, &pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	return buf.String()
}

// newTestBucketStore は通信しないクライアントでバケットハンドルを作る（署名はローカル処理のため実バケットは不要）
func newTestBucketStore(t *testing.T, privateKey string) *firebaseObjectStore {
	t.Helper()
	client, err := storage.NewClient(context.Background(), option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	return &firebaseObjectStore{
		bucket: client.Bucket("test-bucket"),
		sa:     config.ServiceAccount{ClientEmail: "sa@test-project.iam.gserviceaccount.com", PrivateKey: privateKey},
	}
}

func TestFirebaseObjectStore_SignedURL_GeneratesV4SignedURL(t *testing.T) {
	store := newTestBucketStore(t, newTestPrivateKey(t))

	signed, err := store.SignedURL("posts/abc/image.png", time.Now().Add(time.Hour))

	require.NoError(t, err)
	parsed, err := url.Parse(signed)
	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "storage.googleapis.com", parsed.Host)
	assert.Equal(t, "/test-bucket/posts/abc/image.png", parsed.Path)

	query := parsed.Query()
	assert.Equal(t, "GOOG4-RSA-SHA256", query.Get("X-Goog-Algorithm"))
	// 有効秒数は「期限 - 署名時刻」なので、署名までの経過時間だけ 3600 より小さくなることがある
	expiresIn, err := strconv.Atoi(query.Get("X-Goog-Expires"))
	require.NoError(t, err)
	assert.InDelta(t, 3600, expiresIn, 60)
	assert.NotEmpty(t, query.Get("X-Goog-Signature"))
	assert.Contains(t, query.Get("X-Goog-Credential"), "sa@test-project.iam.gserviceaccount.com")
}

func TestFirebaseObjectStore_SignedURL_InvalidPrivateKey(t *testing.T) {
	store := newTestBucketStore(t, "not-a-private-key")

	_, err := store.SignedURL("posts/abc/image.png", time.Now().Add(time.Hour))

	assert.Error(t, err)
}
