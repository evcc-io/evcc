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

-- longduration as duration string (UI-written format), shortduration as legacy nanosecond number
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 2, 'template', 'Duration Test', '', 'Duration Demo Meter', '{"template":"duration-demo","usage":"grid","longduration":"12h","shortduration":15000000000}');

INSERT INTO settings("key", value) VALUES('gridMeter', 'db:1');

COMMIT;
