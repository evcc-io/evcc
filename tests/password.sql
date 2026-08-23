CREATE TABLE IF NOT EXISTS `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

-- password: secret
INSERT INTO settings("key", value) VALUES('adminPassword', '$2a$10$HNLoqiTO5oLwopczA/wcPOebfO79S.hnAA5HOkx5p6o3g5a2E30v2');
