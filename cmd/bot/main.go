package main

import (
	"astro-sarafan/internal/app"
	"flag"
	"log"
	"os"
)

func main() {
	// Флаги для запуска миграций
	runMigrations := flag.Bool("migrate", true, "Запустить миграции базы данных")
	rollbackMigrations := flag.Bool("rollback", false, "Откатить миграции базы данных")
	configPath := flag.String("config", "./config/config.yaml", "Путь к файлу конфигурации")
	verbose := flag.Bool("verbose", false, "Включить подробное логирование")
	flag.Parse()

	// Проверка существования файла конфигурации
	_, err := os.Stat(*configPath)
	if os.IsNotExist(err) {
		log.Fatalf("Конфигурационный файл не найден: %s", *configPath)
	}

	log.Printf("Запуск приложения с параметрами:\n")
	log.Printf("- Конфигурационный файл: %s\n", *configPath)
	log.Printf("- Запуск миграций: %v\n", *runMigrations)
	log.Printf("- Откат миграций: %v\n", *rollbackMigrations)
	log.Printf("- Подробное логирование: %v\n", *verbose)

	// Передаём все параметры в приложение
	if err := app.Run(*configPath, *runMigrations, *rollbackMigrations, *verbose); err != nil {
		log.Fatal(err)
	}
}
