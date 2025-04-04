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
