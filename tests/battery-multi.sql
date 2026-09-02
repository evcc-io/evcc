CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

-- grid discharge switch is still experimental
INSERT INTO `settings`(key, value) VALUES ('experimental', 'true');

-- realistic battery usage thresholds
INSERT INTO `settings`(key, value) VALUES ('prioritySoc', '50');
INSERT INTO `settings`(key, value) VALUES ('bufferSoc', '80');
INSERT INTO `settings`(key, value) VALUES ('bufferStartSoc', '95');
