PRAGMA user_version = 1;

CREATE TABLE IF NOT EXISTS `users` (
	`id` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`name` text NOT NULL,
  `password_hash` text,
	`created_at` datetime DEFAULT (CURRENT_TIMESTAMP) NOT NULL,
	`modified_at` datetime DEFAULT (CURRENT_TIMESTAMP) NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `users_name_unique` ON `users` (`name`);

CREATE TABLE IF NOT EXISTS `sessions` (
  `id` text PRIMARY KEY NOT NULL, -- ULID session ID
  `user_id` integer NOT NULL,
	`created_at` datetime DEFAULT (CURRENT_TIMESTAMP) NOT NULL,
  `last_used_at` datetime,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON UPDATE cascade ON DELETE cascade
)
