-- 管理画面: ユーザーの登録審査・管理者アカウント・パスワードリセット

-- 既存ユーザーは利用中のため approved で追加し、その後の新規登録の既定値を pending にする
ALTER TABLE users ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'approved';
ALTER TABLE users ALTER COLUMN status SET DEFAULT 'pending';
ALTER TABLE users ADD CONSTRAINT chk_users_status CHECK (status IN ('pending', 'approved', 'rejected'));
ALTER TABLE users ADD COLUMN reviewed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN reviewed_by VARCHAR(100);
CREATE INDEX idx_users_status ON users(status);

CREATE TABLE admins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL DEFAULT 'admin' CONSTRAINT chk_admins_role CHECK (role IN ('admin')),
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_by    VARCHAR(100) NOT NULL,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE password_reset_requests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     VARCHAR(20) NOT NULL DEFAULT 'pending'
               CONSTRAINT chk_password_reset_requests_status CHECK (status IN ('pending', 'issued', 'completed')),
    handled_by VARCHAR(100),
    handled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_requests_user_id ON password_reset_requests(user_id);
CREATE INDEX idx_password_reset_requests_status  ON password_reset_requests(status);

CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_id UUID REFERENCES password_reset_requests(id) ON DELETE SET NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at    TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    issued_by  VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id    ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_request_id ON password_reset_tokens(request_id);
