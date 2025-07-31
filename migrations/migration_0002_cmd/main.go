package main

import (
	"fmt"
	"log"

	"github.com/jayydoesdev/airo/bot/migrations/migration_0002"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	success, err := migration_0002.Migrate()
	if err != nil {
		fmt.Println("Migration failed:", err)
		return
	}
	if success {
		fmt.Println("Migration successful")
	}
}
