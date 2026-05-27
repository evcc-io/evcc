CREATE TABLE IF NOT EXISTS `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

-- legacy tariff configuration stored in database
INSERT INTO settings("key", value) VALUES('tariffs', 'currency: EUR');
