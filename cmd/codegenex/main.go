package main

import (
	"fmt"
	"log"
	"os"

	"codegenex/internal/config"
	"codegenex/internal/generator"
	"codegenex/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: codegenex <migration_name> [field:type ...]")
		os.Exit(1)
	}

	migrationName := os.Args[1]
	fields := parser.ParseFields(os.Args[2:])

	cfg := config.GetConfig()
	manager := generator.NewManager(cfg)

	err := manager.GenerateEntity(migrationName, fields)
	if err != nil {
		log.Fatalf("Error generating and saving entity: %v", err)
	}

	fmt.Println("Entity generated successfully.")
}
