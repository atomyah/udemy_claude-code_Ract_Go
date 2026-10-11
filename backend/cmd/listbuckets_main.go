package main

import (
	"context"
	"os"

	"cloud.google.com/go/storage"
	"github.com/atyahara/sns-backend/internal/logger"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	log := logger.L()
	if len(os.Args) < 2 {
		log.Error().Msg("usage: listbuckets <project-id>")
		os.Exit(2)
	}
	credJSON := os.Getenv("FIREBASE_CREDENTIALS_JSON")
	projectID := os.Args[1]
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(credJSON)))
	if err != nil {
		log.Error().Err(err).Msg("storage client init failed")
		return
	}
	it := client.Buckets(ctx, projectID)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("bucket iteration failed")
			return
		}
		log.Info().Str("bucket", attrs.Name).Msg("bucket found")
	}
}
