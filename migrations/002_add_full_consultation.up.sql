ALTER TABLE orders ADD COLUMN IF NOT EXISTS consultation_status VARCHAR(20);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS notification_sent BOOLEAN DEFAULT false;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS unique_client_consultation;
ALTER TABLE users ADD COLUMN IF NOT EXISTS demo_used BOOLEAN DEFAULT false;
ALTER TABLE orders DROP COLUMN IF EXISTS is_full_consultation;
ALTER TABLE orders DROP COLUMN IF EXISTS full_consultation_requested;
ALTER TABLE orders DROP COLUMN IF EXISTS reminder_sent;