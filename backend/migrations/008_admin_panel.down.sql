DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS password_reset_requests;
DROP TABLE IF EXISTS admins;

DROP INDEX IF EXISTS idx_users_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
ALTER TABLE users DROP COLUMN IF EXISTS reviewed_by;
ALTER TABLE users DROP COLUMN IF EXISTS reviewed_at;
ALTER TABLE users DROP COLUMN IF EXISTS status;
