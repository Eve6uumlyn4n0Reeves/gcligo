CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_credentials_updated_at
BEFORE UPDATE ON credentials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_credential_states_updated_at
BEFORE UPDATE ON credential_states
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_configs_updated_at
BEFORE UPDATE ON configs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_usage_stats_updated_at
BEFORE UPDATE ON usage_stats
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
