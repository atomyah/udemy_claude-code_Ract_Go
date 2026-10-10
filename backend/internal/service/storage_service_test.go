package service

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStorageService_Upload_SizeLimits は画像5MB・動画100MBの上限が
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
		{name: "動画100MBちょうどは上限で弾かれない", size: maxVideoSize, maxSize: maxVideoSize, wantTooLarge: false},
		{name: "動画100MB+1バイトは弾かれる", size: maxVideoSize + 1, maxSize: maxVideoSize, wantTooLarge: true},
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

// TestStorageService_SizeLimitConstants は仕様（画像5MB・動画100MB）どおりの
// 定数が設定されていることを検証する
func TestStorageService_SizeLimitConstants(t *testing.T) {
	assert.Equal(t, int64(5*1024*1024), int64(maxImageSize))
	assert.Equal(t, int64(100*1024*1024), int64(maxVideoSize))
}
