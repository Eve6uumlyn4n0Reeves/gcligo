-- PostgreSQL base schema for credentials, configs, and usage tracking
CREATE TABLE IF NOT EXISTS credentials (
    filename     VARCHAR(255) PRIMARY KEY,
    data         JSONB NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credential_states (
    filename   VARCHAR(255) PRIMARY KEY,
    data       JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS configs (
    config_key VARCHAR(255) PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS usage_stats (
    usage_key  VARCHAR(255) NOT NULL,
    field      VARCHAR(255) NOT NULL,
    value      BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (usage_key, field)
);

CREATE INDEX IF NOT EXISTS idx_credentials_filename ON credentials (filename);
CREATE INDEX IF NOT EXISTS idx_credential_states_filename ON credential_states (filename);
CREATE INDEX IF NOT EXISTS idx_usage_stats_usage_key ON usage_stats (usage_key);
CREATE INDEX IF NOT EXISTS idx_usage_stats_field ON usage_stats (field);
CREATE INDEX IF NOT EXISTS idx_configs_key ON configs (config_key);
