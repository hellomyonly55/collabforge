CREATE TABLE IF NOT EXISTS users (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,
 nickname VARCHAR(40) NOT NULL,
 password_hash VARCHAR(60) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 platform_role VARCHAR(16) NOT NULL DEFAULT 'user',
 status VARCHAR(16) NOT NULL DEFAULT 'active',
 session_version BIGINT UNSIGNED NOT NULL DEFAULT 1,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 UNIQUE KEY uk_users_email (email),
 CONSTRAINT chk_users_role CHECK (platform_role IN ('user', 'admin')),
 CONSTRAINT chk_users_status CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
