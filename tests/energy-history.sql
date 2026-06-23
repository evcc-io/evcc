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
  `import` real,
  `export` real
);
CREATE UNIQUE INDEX `meters_meter_ts` ON `meters`(`meter`, `ts`);

-- entities (name = stable id, title = display label)
-- home is a virtual entity; evcc resets its title to "home" on boot
INSERT INTO `entities` (id, `group`, name, title) VALUES (1, 'home', 'home', 'home');
INSERT INTO `entities` (id, `group`, name, title) VALUES (2, 'grid', 'grid', 'Grid');
INSERT INTO `entities` (id, `group`, name, title) VALUES (3, 'battery', 'battery', 'Battery');
INSERT INTO `entities` (id, `group`, name, title) VALUES (4, 'pv', 'solar', 'Solar');
INSERT INTO `entities` (id, `group`, name, title) VALUES (5, 'consumer', 'kitchen', 'Kitchen');
INSERT INTO `entities` (id, `group`, name, title) VALUES (6, 'consumer', 'office', 'Office');
INSERT INTO `entities` (id, `group`, name, title) VALUES (7, 'pv', 'pv-east', 'PV East');
INSERT INTO `entities` (id, `group`, name, title) VALUES (8, 'pv', 'pv-west', 'PV West');
INSERT INTO `entities` (id, `group`, name, title) VALUES (9, 'forecast', 'forecast', 'Forecast');
INSERT INTO `entities` (id, `group`, name, title) VALUES (10, 'battery', 'battery-2', 'Battery 2');
INSERT INTO `entities` (id, `group`, name, title) VALUES (11, 'meter', 'submeter', 'Submeter');

-- =====================================================================
-- API test data: 2026-03-24/25 (existing). Used by the JSON-shape tests.
-- 2026-03-24 22:00, 22:15, 22:30, 22:45 CET (+01:00)  → 4 slots
-- 2026-03-25 00:00, 00:15 CET                         → 2 slots
-- =====================================================================
-- home: import=0.1, export=0 per slot
INSERT INTO `meters` VALUES (1, 1774386000, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774386900, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774387800, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774388700, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774393200, 0.1, 0);
INSERT INTO `meters` VALUES (1, 1774394100, 0.1, 0);

-- grid: import=0.5, export=0.1 per slot  (peak 2 kW bidirectional)
INSERT INTO `meters` VALUES (2, 1774386000, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774386900, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774387800, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774388700, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774393200, 0.5, 0.1);
INSERT INTO `meters` VALUES (2, 1774394100, 0.5, 0.1);

-- =====================================================================
-- UI / chart test data: one local Berlin day per case, 12:00-12:45 CEST.
-- All slots are 15 minutes; energy column is kWh in DB.
-- =====================================================================

-- 2026-04-01 → grid sub-1-kW range. peak 4 W → unit W, symmetric axis.
-- 0.001 kWh × 4 = 0.004 kW = 4 W. Stored values must survive Wh-rounding
-- in the backend (db_history.roundEnergy), so we use kWh values ≥ 0.001.
INSERT INTO `meters` VALUES (2, 1775037600, 0.001, 0.001);
INSERT INTO `meters` VALUES (2, 1775038500, 0.001, 0.001);
INSERT INTO `meters` VALUES (2, 1775039400, 0.001, 0.001);
INSERT INTO `meters` VALUES (2, 1775040300, 0.001, 0.001);

-- 2026-04-02 → grid import-only. peak 2 kW. Axis must still be symmetric
-- because grid is in the hardcoded bidirectional list.
INSERT INTO `meters` VALUES (2, 1775124000, 0.5, 0);
INSERT INTO `meters` VALUES (2, 1775124900, 0.5, 0);
INSERT INTO `meters` VALUES (2, 1775125800, 0.5, 0);
INSERT INTO `meters` VALUES (2, 1775126700, 0.5, 0);

-- 2026-04-03 → battery 2.4 kW peak. niceCeil(2.4) = 3.
-- Alternating slots: charge / discharge / charge / discharge.
INSERT INTO `meters` VALUES (3, 1775210400, 0.6, 0);
INSERT INTO `meters` VALUES (3, 1775211300, 0, 0.5);
INSERT INTO `meters` VALUES (3, 1775212200, 0.6, 0);
INSERT INTO `meters` VALUES (3, 1775213100, 0, 0.5);

-- 2026-04-04 → battery 4.8 kW peak. niceCeil(4.8) = 6 → integer kW labels.
INSERT INTO `meters` VALUES (3, 1775296800, 1.2, 0);
INSERT INTO `meters` VALUES (3, 1775297700, 0, 1.0);
INSERT INTO `meters` VALUES (3, 1775298600, 1.2, 0);
INSERT INTO `meters` VALUES (3, 1775299500, 0, 1.0);

-- 2026-04-05 → PV unidirectional. peak 1.6 kW.
INSERT INTO `meters` VALUES (4, 1775383200, 0.4, 0);
INSERT INTO `meters` VALUES (4, 1775384100, 0.4, 0);
INSERT INTO `meters` VALUES (4, 1775385000, 0.4, 0);
INSERT INTO `meters` VALUES (4, 1775385900, 0.4, 0);

-- 2026-04-06 → PV all-zero day (cloudy). Rows exist but values are 0.
-- Regression: chart group must still render.
INSERT INTO `meters` VALUES (4, 1775469600, 0, 0);
INSERT INTO `meters` VALUES (4, 1775470500, 0, 0);
INSERT INTO `meters` VALUES (4, 1775471400, 0, 0);
INSERT INTO `meters` VALUES (4, 1775472300, 0, 0);

-- 2026-04-07 → consumption breakdown. home = 1.0 kWh total; explicit
-- meters Kitchen 0.4 / Office 0.3; virtual "Others" = home − meters = 0.3.
INSERT INTO `meters` VALUES (1, 1775556000, 0.25, 0);
INSERT INTO `meters` VALUES (1, 1775556900, 0.25, 0);
INSERT INTO `meters` VALUES (1, 1775557800, 0.25, 0);
INSERT INTO `meters` VALUES (1, 1775558700, 0.25, 0);
INSERT INTO `meters` VALUES (5, 1775556000, 0.1, 0);
INSERT INTO `meters` VALUES (5, 1775556900, 0.1, 0);
INSERT INTO `meters` VALUES (5, 1775557800, 0.1, 0);
INSERT INTO `meters` VALUES (5, 1775558700, 0.1, 0);
INSERT INTO `meters` VALUES (6, 1775556000, 0.075, 0);
INSERT INTO `meters` VALUES (6, 1775556900, 0.075, 0);
INSERT INTO `meters` VALUES (6, 1775557800, 0.075, 0);
INSERT INTO `meters` VALUES (6, 1775558700, 0.075, 0);

-- 2026-05-02 → multi-entity PV with stacked peak 7.2 kW. Two PV entities
-- (east + west) plus a forecast overlay. axisPeak comes from the stacked
-- per-slot sum, niceCeil → 8 → integer kW labels, echarts auto-range.
INSERT INTO `meters` VALUES (7, 1777687200, 0.05, 0); -- 04:00
INSERT INTO `meters` VALUES (7, 1777694400, 0.30, 0); -- 06:00
INSERT INTO `meters` VALUES (7, 1777701600, 0.60, 0); -- 08:00
INSERT INTO `meters` VALUES (7, 1777708800, 0.80, 0); -- 10:00
INSERT INTO `meters` VALUES (7, 1777716000, 1.00, 0); -- 12:00
INSERT INTO `meters` VALUES (7, 1777723200, 0.70, 0); -- 14:00
INSERT INTO `meters` VALUES (7, 1777730400, 0.30, 0); -- 16:00
INSERT INTO `meters` VALUES (7, 1777737600, 0.05, 0); -- 18:00
INSERT INTO `meters` VALUES (8, 1777687200, 0, 0);
INSERT INTO `meters` VALUES (8, 1777694400, 0.10, 0);
INSERT INTO `meters` VALUES (8, 1777701600, 0.40, 0);
INSERT INTO `meters` VALUES (8, 1777708800, 0.60, 0);
INSERT INTO `meters` VALUES (8, 1777716000, 0.80, 0);
INSERT INTO `meters` VALUES (8, 1777723200, 1.00, 0);
INSERT INTO `meters` VALUES (8, 1777730400, 0.80, 0);
INSERT INTO `meters` VALUES (8, 1777737600, 0.30, 0);
INSERT INTO `meters` VALUES (9, 1777687200, 0.05, 0);
INSERT INTO `meters` VALUES (9, 1777694400, 0.40, 0);
INSERT INTO `meters` VALUES (9, 1777701600, 1.00, 0);
INSERT INTO `meters` VALUES (9, 1777708800, 1.40, 0);
INSERT INTO `meters` VALUES (9, 1777716000, 1.80, 0);
INSERT INTO `meters` VALUES (9, 1777723200, 1.70, 0);
INSERT INTO `meters` VALUES (9, 1777730400, 1.10, 0);
INSERT INTO `meters` VALUES (9, 1777737600, 0.35, 0);

-- 2026-04-08 → two stacked batteries. Charge 1.2 kW each (stacked 2.4 kW),
-- discharge 0.8 kW each (stacked -1.6 kW). niceCeil(2.4) = 3.
INSERT INTO `meters` VALUES (3, 1775642400, 0.3, 0);
INSERT INTO `meters` VALUES (3, 1775643300, 0, 0.2);
INSERT INTO `meters` VALUES (3, 1775644200, 0.3, 0);
INSERT INTO `meters` VALUES (3, 1775645100, 0, 0.2);
INSERT INTO `meters` VALUES (10, 1775642400, 0.3, 0);
INSERT INTO `meters` VALUES (10, 1775643300, 0, 0.2);
INSERT INTO `meters` VALUES (10, 1775644200, 0.3, 0);
INSERT INTO `meters` VALUES (10, 1775645100, 0, 0.2);

-- =====================================================================
-- Month-aggregation case: 2026-06, three days with battery bidirectional
-- data. Daily totals ~1.4 / 1.0 kWh → limit rounds to 2 → 1-decimal kWh.
-- =====================================================================
INSERT INTO `meters` VALUES (3, 1781517600, 1.4, 1.0);
INSERT INTO `meters` VALUES (3, 1781604000, 1.4, 1.0);
INSERT INTO `meters` VALUES (3, 1781690400, 1.4, 1.0);

-- 2026-04-09 → additional meter (ext) standalone chart. Single entity 1.2 kWh,
-- not home-combined, so no virtual "Others" series.
INSERT INTO `meters` VALUES (11, 1775728800, 0.3, 0);
INSERT INTO `meters` VALUES (11, 1775729700, 0.3, 0);
INSERT INTO `meters` VALUES (11, 1775730600, 0.3, 0);
INSERT INTO `meters` VALUES (11, 1775731500, 0.3, 0);

-- 2026-04-10 → additional meter with export. Import peak 2 kW, export 0.4 kW.
-- Negative values must flip the axis to symmetric (bidirectional) ±2 kW.
INSERT INTO `meters` VALUES (11, 1775815200, 0.5, 0.1);
INSERT INTO `meters` VALUES (11, 1775816100, 0.5, 0.1);
INSERT INTO `meters` VALUES (11, 1775817000, 0.5, 0.1);
INSERT INTO `meters` VALUES (11, 1775817900, 0.5, 0.1);
