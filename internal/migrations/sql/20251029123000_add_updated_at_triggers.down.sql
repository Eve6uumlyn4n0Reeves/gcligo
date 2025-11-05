DROP TRIGGER IF EXISTS trg_usage_stats_updated_at ON usage_stats;
DROP TRIGGER IF EXISTS trg_configs_updated_at ON configs;
DROP TRIGGER IF EXISTS trg_credential_states_updated_at ON credential_states;
DROP TRIGGER IF EXISTS trg_credentials_updated_at ON credentials;

DROP FUNCTION IF EXISTS set_updated_at();
