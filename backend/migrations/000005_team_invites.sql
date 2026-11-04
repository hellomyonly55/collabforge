CREATE TABLE IF NOT EXISTS team_invites (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 team_id BIGINT UNSIGNED NOT NULL,
 token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 max_uses INT UNSIGNED NOT NULL,
 used_count INT UNSIGNED NOT NULL DEFAULT 0,
 revoked_at DATETIME(6) NULL,
 created_by BIGINT UNSIGNED NOT NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 UNIQUE KEY uk_invite_hash (token_hash),
 KEY idx_invites_team (team_id,id),
 CONSTRAINT fk_invites_team FOREIGN KEY (team_id) REFERENCES teams(id),
 CONSTRAINT fk_invites_creator FOREIGN KEY (created_by) REFERENCES users(id),
 CONSTRAINT chk_invite_uses CHECK (max_uses > 0 AND used_count <= max_uses)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
