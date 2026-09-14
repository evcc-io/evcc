CREATE TABLE IF NOT EXISTS `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

INSERT INTO settings("key", "value") VALUES ("batteryGridChargeLimit", "0.12");
INSERT INTO settings("key", "value") VALUES ("lp1.smartCostLimit", "0.12");
