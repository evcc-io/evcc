CREATE TABLE `entities` (
  `id` integer,
  `group` text,
  `name` text,
  `title` text,
  PRIMARY KEY (`id`)
);

CREATE TABLE `meters` (
  `meter` integer,
  `ts` integer,
  `energy` real,
  `return_energy` real
);

INSERT INTO `entities` (id, `group`, name, title) VALUES (1, 'grid', 'grid', 'grid');
INSERT INTO `entities` (id, `group`, name, title) VALUES (2, 'battery', 'battery', 'Battery');
INSERT INTO `entities` (id, `group`, name, title) VALUES (3, 'pv', 'solar', 'Solar');
INSERT INTO `entities` (id, `group`, name, title) VALUES (4, 'loadpoint', 'lp-1', 'Carport');

-- 2 hours ago, relative since the stats windows are rolling: pv 1, battery 1 and grid 2 → carport 4
INSERT INTO `meters` SELECT 3, CAST(strftime('%s', 'now') AS INTEGER) / 900 * 900 - 7200, 1, 0;
INSERT INTO `meters` SELECT 2, CAST(strftime('%s', 'now') AS INTEGER) / 900 * 900 - 7200, 0, 1;
INSERT INTO `meters` SELECT 1, CAST(strftime('%s', 'now') AS INTEGER) / 900 * 900 - 7200, 2, 0;
INSERT INTO `meters` SELECT 4, CAST(strftime('%s', 'now') AS INTEGER) / 900 * 900 - 7200, 4, 0;
