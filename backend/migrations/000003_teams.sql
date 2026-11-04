CREATE TABLE IF NOT EXISTS teams (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 name VARCHAR(80) NOT NULL,
 description VARCHAR(2000) NOT NULL DEFAULT '',
 owner_id BIGINT UNSIGNED NOT NULL,
 archived_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_teams_owner FOREIGN KEY (owner_id) REFERENCES users(id),
 KEY idx_teams_created (created_at,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
