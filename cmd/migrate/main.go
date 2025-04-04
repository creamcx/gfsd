package main

import (
	"astro-sarafan/internal/config"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

const migrationSchema = `
-- Создание таблицы для хранения пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    full_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создание таблицы для хранения заказов
CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(20) PRIMARY KEY,
    client_id BIGINT NOT NULL REFERENCES users(chat_id),
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    taken_at TIMESTAMP,
    astrologer_id BIGINT,
    astrologer_name VARCHAR(255),
    CONSTRAINT unique_client_consultation UNIQUE (client_id) -- Ограничение: один клиент - одна консультация
);

-- Индексы для ускорения запросов
CREATE INDEX IF NOT EXISTS idx_users_chat_id ON users(chat_id);
CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders(client_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
`

func main() {
	// Загружаем конфигурацию
	cfg, err := config.NewConfig("config.yaml")
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
