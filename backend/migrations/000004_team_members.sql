CREATE TABLE IF NOT EXISTS team_members (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 team_id BIGINT UNSIGNED NOT NULL,
 user_id BIGINT UNSIGNED NOT NULL,
 role VARCHAR(16) NOT NULL,
 joined_at DATETIME(6) NOT NULL,
 removed_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 owner_guard TINYINT GENERATED ALWAYS AS (CASE WHEN role = 'owner' AND removed_at IS NULL THEN 1 ELSE NULL END) STORED,
 UNIQUE KEY uk_team_member (team_id,user_id),
 UNIQUE KEY uk_team_owner (team_id,owner_guard),
 KEY idx_members_user (user_id,removed_at,team_id),
 CONSTRAINT fk_members_team FOREIGN KEY (team_id) REFERENCES teams(id),
 CONSTRAINT fk_members_user FOREIGN KEY (user_id) REFERENCES users(id),
 CONSTRAINT chk_member_role CHECK (role IN ('owner','admin','member'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
