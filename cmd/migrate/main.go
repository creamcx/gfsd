package main

import (
	"astro-sarafan/internal/config"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

const migrationScript = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    full_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    referral_code VARCHAR(20) UNIQUE
);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(20) PRIMARY KEY,
    client_id BIGINT NOT NULL REFERENCES users(chat_id),
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    taken_at TIMESTAMP,
    consultation_started_at TIMESTAMP,
    astrologer_id BIGINT,
    astrologer_name VARCHAR(255),
    referrer_id BIGINT,
    referrer_name VARCHAR(255),
    reminder_sent BOOLEAN DEFAULT false,
    CONSTRAINT unique_client_consultation UNIQUE (client_id)
);

-- Индексы для ускорения запросов
CREATE INDEX IF NOT EXISTS idx_users_chat_id ON users(chat_id);
CREATE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);
CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders(client_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_referrer_id ON orders(referrer_id);
CREATE INDEX IF NOT EXISTS idx_orders_consultation_started_at ON orders(consultation_started_at);
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
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
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

	// Выполняем миграцию
	_, err = db.Exec(migrationScript)
	if err != nil {
		log.Fatalf("Ошибка выполнения миграции: %v", err)
	}

	fmt.Println("Миграция базы данных успешно выполнена")
	os.Exit(0)
}
