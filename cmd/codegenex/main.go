package main

import (
	"fmt"
	"os"

	"codegenex/internal/config"
	"codegenex/internal/generator"
	"codegenex/internal/parser"
)

func main() {
	// Загружаем конфигурацию при запуске приложения
	config.GetConfig()

	if len(os.Args) < 2 {
		fmt.Println("Usage: codegenex <migration_name> [field:type ...]")
		os.Exit(1)
	}

	migrationName := os.Args[1]
	fields := parser.ParseFields(os.Args[2:])

	migration, err := generator.GenerateMigrationAndModel(migrationName, fields)
	if err != nil {
		fmt.Printf("Error generating migration and model: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(migration)
}
