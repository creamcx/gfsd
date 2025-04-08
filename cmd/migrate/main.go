package main

import (
	"astro-sarafan/internal/config"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

const migrationSchema = `ALTER TABLE orders 
ADD COLUMN reminder_sent BOOLEAN DEFAULT false;
`

func main() {
	// Загружаем конфигурацию
	cfg, err := config.NewConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Создаем DSN
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки подключения к базе данных: %v", err)
	}

	fmt.Println("Успешное подключение к базе данных")

	// Выполняем миграцию
	_, err = db.Exec(migrationSchema)
	if err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}

	fmt.Println("Миграция успешно выполнена")
	os.Exit(0)
}
