CREATE TABLE IF NOT EXISTS activity_logs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 team_id BIGINT UNSIGNED NOT NULL,
 project_id BIGINT UNSIGNED NULL,
 actor_id BIGINT UNSIGNED NOT NULL,
 action VARCHAR(64) NOT NULL,
 target_type VARCHAR(32) NOT NULL,
 target_id BIGINT UNSIGNED NOT NULL,
 metadata JSON NOT NULL,
 created_at DATETIME(6) NOT NULL,
 KEY idx_activity_team (team_id,id),
 KEY idx_activity_project (project_id,id),
 KEY idx_activity_time (team_id,created_at),
 CONSTRAINT fk_activity_team FOREIGN KEY (team_id) REFERENCES teams(id),
 CONSTRAINT fk_activity_actor FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
