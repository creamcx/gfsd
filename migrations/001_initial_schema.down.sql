-- Удаление индексов
DROP INDEX IF EXISTS idx_users_chat_id;
DROP INDEX IF EXISTS idx_users_referral_code;
DROP INDEX IF EXISTS idx_orders_client_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_referrer_id;
DROP INDEX IF EXISTS idx_orders_consultation_started_at;

-- Удаление таблиц
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;