CREATE TABLE IF NOT EXISTS projects (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 team_id BIGINT UNSIGNED NOT NULL,
 name VARCHAR(100) NOT NULL,
 description VARCHAR(2000) NOT NULL DEFAULT '',
 created_by BIGINT UNSIGNED NOT NULL,
 archived_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 UNIQUE KEY uk_projects_team_id (team_id,id),
 KEY idx_projects_team_status (team_id,archived_at,id),
 CONSTRAINT fk_projects_team FOREIGN KEY (team_id) REFERENCES teams(id),
 CONSTRAINT fk_projects_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
