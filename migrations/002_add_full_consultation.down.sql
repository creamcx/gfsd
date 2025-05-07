-- Новая миграция: 002_add_full_consultation.down.sql
ALTER TABLE orders DROP COLUMN IF EXISTS consultation_status;
ALTER TABLE orders DROP COLUMN IF EXISTS notification_sent;