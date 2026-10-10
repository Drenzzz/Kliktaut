-- =============================================
-- TABLE: links
-- =============================================
CREATE TABLE links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_url    TEXT NOT NULL,
    short_code      VARCHAR(50) UNIQUE NOT NULL,
    title           VARCHAR(255),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    click_count     BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_links_expires_at ON links(expires_at);
CREATE INDEX idx_links_created_at ON links(created_at DESC);

-- =============================================
-- TABLE: clicks
-- =============================================
CREATE TABLE clicks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id         UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    ip_address      INET,
    user_agent      TEXT,
    referrer        TEXT,
    country         VARCHAR(10),
    device_type     VARCHAR(30),
    browser         VARCHAR(100),
    os              VARCHAR(100),
    clicked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clicks_link_id ON clicks(link_id);
CREATE INDEX idx_clicks_clicked_at ON clicks(clicked_at DESC);
CREATE INDEX idx_clicks_device_type ON clicks(device_type);
CREATE INDEX idx_clicks_link_id_clicked_at ON clicks(link_id, clicked_at DESC);
