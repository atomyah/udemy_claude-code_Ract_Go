CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    handle        VARCHAR(50)  UNIQUE NOT NULL,
    display_name  VARCHAR(50)  NOT NULL,
    avatar_url    VARCHAR(500),
    banner_url    VARCHAR(500),
    bio           VARCHAR(160),
    location      VARCHAR(30),
    website_url   VARCHAR(100),
    birthday      DATE,
    theme         VARCHAR(10)  NOT NULL DEFAULT 'light',
    role          VARCHAR(10)  NOT NULL DEFAULT 'user',
    is_suspended  BOOLEAN      NOT NULL DEFAULT false,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_handle ON users(handle);
CREATE INDEX idx_users_email  ON users(email);
