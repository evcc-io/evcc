DROP TABLE IF EXISTS `meters`;
DROP TABLE IF EXISTS `entities`;

CREATE TABLE `entities` (
  `id` integer,
  `group` text,
  `name` text,
  PRIMARY KEY (`id`)
);
CREATE UNIQUE INDEX `entities_group_name` ON `entities`(`group`, `name`);

CREATE TABLE `meters` (
  `meter` integer,
  `ts` integer,
  `import` real,
  `export` real
);
CREATE UNIQUE INDEX `meters_meter_ts` ON `meters`(`meter`, `ts`);

-- entities
INSERT INTO `entities` VALUES (1, 'virtual', 'home');
INSERT INTO `entities` VALUES (2, 'grid', 'grid');

-- meter data: 6 slots per entity spanning midnight (local time +01:00)
-- 2026-03-24: 22:00, 22:15, 22:30, 22:45 (4 slots)
-- 2026-03-25: 00:00, 00:15 (2 slots)

-- home (id=1): import=0.1, export=0 per slot
INSERT INTO `meters` VALUES (1, 1774386000, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774386900, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774387800, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774388700, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774393200, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774394100, 0.1, 0);

-- grid (id=2): import=0.5, export=0.1 per slot
INSERT INTO `meters` VALUES (2, 1774386000, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774386900, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774387800, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774388700, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774393200, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774394100, 0.5, 0.1);
