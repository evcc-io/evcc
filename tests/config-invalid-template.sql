BEGIN;

CREATE TABLE `configs` (
    `id` integer PRIMARY KEY AUTOINCREMENT
  , `class` integer
  , `type` text
  , `title` text
  , `icon` text
  , `product` text
  , `value` text
);
CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 2, 'template', '', '', 'Demo meter', '{"maxacpower":"0","power_old":222,"template":"demo-meter","usage":"grid"}');
INSERT INTO settings("key", value) VALUES('gridMeter', 'db:1');

COMMIT;