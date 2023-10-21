DROP TABLE IF EXISTS `sessions`;
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

INSERT INTO `sessions` VALUES (1, datetime('now',   '-1 day'),datetime('now',   '-0 day'),'Carport',NULL,'e-Golf',NULL,NULL,40,NULL, 50, 8,0.2,10,null);
INSERT INTO `sessions` VALUES (2, datetime('now',  '-11 day'),datetime('now',  '-10 day'),'Carport',NULL,'e-Golf',NULL,NULL,10,NULL,100, 1,0.1, 0,null);
INSERT INTO `sessions` VALUES (3, datetime('now', '-101 day'),datetime('now', '-100 day'),'Carport',NULL,'e-Golf',NULL,NULL,50,NULL,  0,15,0.3,20,null);
INSERT INTO `sessions` VALUES (4, datetime('now', '-901 day'),datetime('now', '-900 day'),'Carport',NULL,'e-Golf',NULL,NULL,40,NULL,100, 3,0.1, 0,null);
