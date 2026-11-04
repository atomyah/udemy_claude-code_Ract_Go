package config

import (
	"context"
	"encoding/json"
	"errors"
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

// ServiceAccount は署名付きURLの生成に使うサービスアカウントの情報
type ServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// ParseServiceAccount は FIREBASE_CREDENTIALS_JSON の文字列からサービスアカウントの情報を取り出す。
// エラーメッセージがログに残るため、JSON のパースエラーをそのまま返さず値を含めない
func ParseServiceAccount(credentialsJSON string) (ServiceAccount, error) {
	var sa ServiceAccount
	if err := json.Unmarshal([]byte(credentialsJSON), &sa); err != nil {
		return ServiceAccount{}, errors.New("FIREBASE_CREDENTIALS_JSON を JSON として読み取れません")
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return ServiceAccount{}, errors.New("FIREBASE_CREDENTIALS_JSON に client_email / private_key がありません")
	}
	return sa, nil
}
