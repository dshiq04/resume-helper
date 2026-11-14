-- ResumeTrail initialization script for MySQL 8.0+
-- Usage:
--   mysql -u root -p < init.mysql.sql
--
-- The application also runs GORM AutoMigrate at startup. This script exists
-- for manual provisioning or for inspecting the generated schema.

CREATE DATABASE IF NOT EXISTS resume_helper
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE resume_helper;

CREATE TABLE IF NOT EXISTS applications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  category VARCHAR(32) NOT NULL,
  company_name VARCHAR(255) NOT NULL,
  link TEXT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'screening',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  applied_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_applications_category (category),
  KEY idx_applications_company_name (company_name),
  KEY idx_applications_status (status),
  KEY idx_applications_pinned (pinned),
  KEY idx_applications_applied_at (applied_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS settings (
  `key` VARCHAR(64) NOT NULL,
  `value` TEXT NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS resumes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  tags TEXT NULL,
  file_name VARCHAR(255) NOT NULL,
  stored_name VARCHAR(255) NOT NULL,
  file_type VARCHAR(16) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  encrypted BOOLEAN NOT NULL DEFAULT FALSE,
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_resumes_stored_name (stored_name),
  KEY idx_resumes_name (name),
  KEY idx_resumes_pinned (pinned)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci;
