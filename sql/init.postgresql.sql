-- ResumeTrail initialization script for PostgreSQL 13+
-- Usage:
--   createdb resume_helper
--   psql -d resume_helper -f init.postgresql.sql
--
-- The application also runs GORM AutoMigrate at startup. This script exists
-- for manual provisioning or for inspecting the generated schema.

CREATE TABLE IF NOT EXISTS applications (
  id BIGSERIAL PRIMARY KEY,
  category VARCHAR(32) NOT NULL,
  company_name VARCHAR(255) NOT NULL,
  link TEXT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'screening',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_applications_category
  ON applications (category);
CREATE INDEX IF NOT EXISTS idx_applications_company_name
  ON applications (company_name);
CREATE INDEX IF NOT EXISTS idx_applications_status
  ON applications (status);
CREATE INDEX IF NOT EXISTS idx_applications_pinned
  ON applications (pinned);

CREATE TABLE IF NOT EXISTS settings (
  key VARCHAR(64) NOT NULL,
  value TEXT NULL,
  updated_at TIMESTAMPTZ NULL,
  PRIMARY KEY (key)
);
