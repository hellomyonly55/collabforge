CREATE TABLE IF NOT EXISTS platform_audit_logs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 actor_id BIGINT UNSIGNED NOT NULL,
 action VARCHAR(64) NOT NULL,
 target_type VARCHAR(32) NOT NULL,
 target_id BIGINT UNSIGNED NOT NULL,
 metadata JSON NOT NULL,
 created_at DATETIME(6) NOT NULL,
 KEY idx_platform_audit_actor (actor_id,id),
 KEY idx_platform_audit_time (created_at,id),
 CONSTRAINT fk_platform_audit_actor FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
