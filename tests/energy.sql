CREATE TABLE `entities` (
  `id` integer,
  `group` text,
  `name` text,
  `title` text,
  PRIMARY KEY (`id`)
);
CREATE UNIQUE INDEX `entities_group_name` ON `entities`(`group`, `name`);

CREATE TABLE `meters` (
  `meter` integer,
  `ts` integer,
  `energy` real,
  `return_energy` real
);
CREATE UNIQUE INDEX `meters_meter_ts` ON `meters`(`meter`, `ts`);

CREATE TABLE `tariffs` (
  `ts` integer,
  `grid` real,
  `feedin` real,
  `co2` real,
  `temperature` real
);
CREATE UNIQUE INDEX `idx_tariffs_timestamp` ON `tariffs`(`ts`);

INSERT INTO `entities` (id, `group`, name, title) VALUES (1, 'home', 'home', 'home');
INSERT INTO `entities` (id, `group`, name, title) VALUES (2, 'grid', 'grid', 'grid');
INSERT INTO `entities` (id, `group`, name, title) VALUES (3, 'battery', 'battery', 'Battery');
INSERT INTO `entities` (id, `group`, name, title) VALUES (4, 'pv', 'solar', 'Solar');
INSERT INTO `entities` (id, `group`, name, title) VALUES (5, 'loadpoint', 'carport', 'Carport');
INSERT INTO `entities` (id, `group`, name, title) VALUES (6, 'forecast', 'forecast', 'forecast');
INSERT INTO `entities` (id, `group`, name, title) VALUES (7, 'meter', 'pool', 'Pool');

-- 2026-09-15 12:00 and 12:15 CEST (+02:00): sunny, export and battery charging
-- pv 2 → home 0.5, battery 0.5, loadpoint 0.5, export 0.5
INSERT INTO `meters` VALUES (4, 1789466400, 2, 0);
INSERT INTO `meters` VALUES (7, 1789466400, 0.3, 0);
INSERT INTO `meters` VALUES (6, 1789466400, 1.6, 0);
INSERT INTO `meters` VALUES (1, 1789466400, 0.5, 0);
INSERT INTO `meters` VALUES (3, 1789466400, 0.5, 0);
INSERT INTO `meters` VALUES (5, 1789466400, 0.5, 0);
INSERT INTO `meters` VALUES (2, 1789466400, 0, 0.5);
INSERT INTO `meters` VALUES (4, 1789467300, 2, 0);
INSERT INTO `meters` VALUES (6, 1789467300, 1.6, 0);
INSERT INTO `meters` VALUES (1, 1789467300, 0.5, 0);
INSERT INTO `meters` VALUES (3, 1789467300, 0.5, 0);
INSERT INTO `meters` VALUES (5, 1789467300, 0.5, 0);
INSERT INTO `meters` VALUES (2, 1789467300, 0, 0.5);

-- 2026-09-15 22:00 and 22:15 CEST: night, battery discharge and grid import
-- battery 0.5 → home 0.5, grid 1 → loadpoint 1
INSERT INTO `meters` VALUES (3, 1789502400, 0, 0.5);
INSERT INTO `meters` VALUES (1, 1789502400, 0.5, 0);
INSERT INTO `meters` VALUES (5, 1789502400, 1, 0);
INSERT INTO `meters` VALUES (2, 1789502400, 1, 0);
INSERT INTO `meters` VALUES (3, 1789503300, 0, 0.5);
INSERT INTO `meters` VALUES (1, 1789503300, 0.5, 0);
INSERT INTO `meters` VALUES (5, 1789503300, 1, 0);
INSERT INTO `meters` VALUES (2, 1789503300, 1, 0);

-- tariffs for all four slots: grid 0.30 (night 0.25 / 0.35), feed-in 0.10, co2 400 g/kWh
INSERT INTO `tariffs` VALUES (1789466400, 0.3, 0.1, 400, NULL);
INSERT INTO `tariffs` VALUES (1789467300, 0.3, 0.1, 400, NULL);
INSERT INTO `tariffs` VALUES (1789502400, 0.25, 0.1, 400, NULL);
INSERT INTO `tariffs` VALUES (1789503300, 0.35, 0.1, 400, NULL);
