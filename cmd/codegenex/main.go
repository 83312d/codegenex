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
	if len(os.Args) < 3 {
		fmt.Println("Usage: codegenex <entity_name> <action> [field:type:options ...]")
		os.Exit(1)
	}

	entityName := os.Args[1]
	action := parser.ParseAction(os.Args[2])
	fields := parser.ParseFields(os.Args[3:])

	cfg := config.GetConfig()
	manager := generator.NewManager(cfg)

	err := manager.GenerateEntity(entityName, action, fields)
	if err != nil {
		log.Fatalf("Error generating and saving entity: %v", err)
	}

	fmt.Println("Entity updated successfully.")
}
