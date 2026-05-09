ALTER TABLE email_provider_configs
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
