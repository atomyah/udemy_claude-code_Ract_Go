package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	credJSON := os.Getenv("FIREBASE_CREDENTIALS_JSON")
	projectID := os.Args[1]
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(credJSON)))
	if err != nil {
		fmt.Println("client err:", err)
		return
	}
	it := client.Buckets(ctx, projectID)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println("iter err:", err)
			return
		}
		fmt.Println("bucket:", attrs.Name)
	}
}
