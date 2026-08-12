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

INSERT INTO configs(class, type, value) VALUES(3, 'template', '{"template":"offline","title":"Grey Car","mode":"now"}');

COMMIT;
