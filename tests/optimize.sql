CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

-- optimizer is still experimental
INSERT INTO `settings`(key, value) VALUES ('experimental', 'true');
INSERT INTO `settings`(key, value) VALUES ('optimizer', 'true');
