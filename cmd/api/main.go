package main

import (
	"backend-go/internal/db"
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}

	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()
	fmt.Print("Connected to db successfully")
}
