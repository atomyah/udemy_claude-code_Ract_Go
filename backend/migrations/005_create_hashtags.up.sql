CREATE TABLE hashtags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE post_hashtags (
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    hashtag_id UUID NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY(post_id, hashtag_id)
);

CREATE INDEX idx_hashtags_name         ON hashtags(name);
CREATE INDEX idx_post_hashtags_hashtag ON post_hashtags(hashtag_id);
