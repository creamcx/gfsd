-- Добавляем столбец для отслеживания нажатия кнопки в таблицу orders
ALTER TABLE orders ADD COLUMN IF NOT EXISTS button_pressed BOOLEAN DEFAULT FALSE;

-- Добавляем столбец для хранения времени отправки напоминания
ALTER TABLE orders ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP;

-- Добавляем столбец для хранения URL PDF с консультацией
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pdf_url VARCHAR(255);

-- Добавляем столбец для хранения времени отправки PDF с консультацией
ALTER TABLE orders ADD COLUMN IF NOT EXISTS pdf_sent_at TIMESTAMP;

-- Создаем индекс для ускорения поиска заказов, которым нужно отправить напоминание
CREATE INDEX IF NOT EXISTS idx_orders_taken_at ON orders(taken_at);