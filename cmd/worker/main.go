package main

import (
	"fmt"
	"log"
	"os"

	"github.com/coolycow/gophprofile/internal/config"
	"github.com/coolycow/gophprofile/internal/service"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	// Вывод информации о сборке при старте
	fmt.Fprintf(os.Stdout, "Build version: %s\n", service.OrNA(buildVersion))
	fmt.Fprintf(os.Stdout, "Build date: %s\n", service.OrNA(buildDate))
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", service.OrNA(buildCommit))

	// Инициализируем настройки (приоритет: окружение, флаги, дефолт)
	cfg, err := config.InitConfigWorker()

	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Выводим настройки в лог
	cfg.PrintWorkerConfig()
}
