package config

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// NewFirebaseApp はFirebase Admin SDKアプリを初期化する。
// Storageアップロード（storage_service.go）とGoogle OAuthのIDトークン検証（auth_service.go）で共有する。
func NewFirebaseApp(cfg *Config) (*firebase.App, error) {
	app, err := firebase.NewApp(context.Background(),
		&firebase.Config{StorageBucket: cfg.FirebaseStorageBucket},
		option.WithCredentialsJSON([]byte(cfg.FirebaseCredentials)),
	)
	if err != nil {
		return nil, fmt.Errorf("firebase app init: %w", err)
	}
	return app, nil
}
