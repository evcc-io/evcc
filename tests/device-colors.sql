CREATE TABLE `sessions` (
  `id` integer,
  `created` datetime,
  `finished` datetime,
  `loadpoint` text,
  `identifier` text,
  `vehicle` text,
  `meter_start_kwh` real,
  `meter_end_kwh` real,
  `charged_kwh` real,
  `odometer` real,
  `solar_percentage` real,
  `price` real,
  `price_per_kwh` real,
  `co2_per_kwh` real,
  `charge_duration` integer,
  PRIMARY KEY (`id`)
);

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

CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

CREATE TABLE `configs` (
    `id` integer PRIMARY KEY AUTOINCREMENT
  , `class` integer
  , `type` text
  , `title` text
  , `icon` text
  , `product` text
  , `value` text
);

-- Honda charged on each loadpoint, Garage has more energy than Carport
INSERT INTO `sessions` VALUES (1, '2026-05-01 08:00:00.0+02:00', '2026-05-01 09:00:00.0+02:00', 'Garage',  NULL, 'Honda', NULL, NULL, 8.0, NULL, 50, 1.0, 0.2, 100, 3600000000000);
INSERT INTO `sessions` VALUES (2, '2026-05-02 08:00:00.0+02:00', '2026-05-02 09:00:00.0+02:00', 'Carport', NULL, 'Honda', NULL, NULL, 4.0, NULL, 50, 0.5, 0.2,  50, 3600000000000);

-- db-backed consumer meter (Dishwasher, usage=charge) → "Consumption" chart
INSERT INTO `configs`(id, class, type, title, icon, product, value)
  VALUES (1, 2, 'template', 'Dishwasher', '', 'Demo meter',
          '{"power":"100","template":"demo-meter","usage":"charge"}');

-- db-backed ext meter (alternative grid meter, usage=grid) → "Additional meters" chart
INSERT INTO `configs`(id, class, type, title, icon, product, value)
  VALUES (2, 2, 'template', 'Alternative grid', '', 'Demo meter',
          '{"power":"100","template":"demo-meter","usage":"grid"}');

-- assign consumer + ext refs, enable experimental
INSERT INTO `settings`(key, value) VALUES ('consumerMeters', 'db:1');
INSERT INTO `settings`(key, value) VALUES ('extMeters',      'db:2');
INSERT INTO `settings`(key, value) VALUES ('experimental',   'true');

-- history entities matching evcc internal names
INSERT INTO `entities` VALUES (1, 'loadpoint', 'lp-1');
INSERT INTO `entities` VALUES (2, 'loadpoint', 'lp-2');
INSERT INTO `entities` VALUES (3, 'home',      'home');
INSERT INTO `entities` VALUES (4, 'consumer',  'db:1');
INSERT INTO `entities` VALUES (5, 'meter',     'db:2');

-- one 15-min slot at 2026-05-15 12:00:00+02:00 (ts=1778839200) per entity
INSERT INTO `meters` VALUES (1, 1778839200, 2.0, 0); -- lp-1 (Garage)
INSERT INTO `meters` VALUES (2, 1778839200, 1.0, 0); -- lp-2 (Carport)
INSERT INTO `meters` VALUES (3, 1778839200, 1.0, 0); -- home
INSERT INTO `meters` VALUES (4, 1778839200, 0.5, 0); -- Dishwasher (consumer)
INSERT INTO `meters` VALUES (5, 1778839200, 0.3, 0); -- Alternative grid (ext)
