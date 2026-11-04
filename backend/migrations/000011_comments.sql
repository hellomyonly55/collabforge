CREATE TABLE IF NOT EXISTS comments (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 task_id BIGINT UNSIGNED NOT NULL,
 author_id BIGINT UNSIGNED NOT NULL,
 content VARCHAR(2000) NOT NULL,
 version BIGINT UNSIGNED NOT NULL DEFAULT 1,
 deleted_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 KEY idx_comments_task (task_id,deleted_at,id),
 CONSTRAINT fk_comments_task FOREIGN KEY (task_id) REFERENCES tasks(id),
 CONSTRAINT fk_comments_author FOREIGN KEY (author_id) REFERENCES users(id),
 CONSTRAINT chk_comments_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
