CREATE TABLE IF NOT EXISTS `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

-- legacy messenger configuration stored in database
INSERT INTO settings("key", value) VALUES('messaging', 'events:
  start:
    title: Charge started
    msg: Started charging');
