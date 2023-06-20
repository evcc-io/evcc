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
  PRIMARY KEY (`id`)
);

INSERT INTO `sessions` VALUES (1,'2023-05-02 08:00:00.0+02:00','2023-05-02 12:00:00.0+02:00','Carport',NULL,'blauer e-Golf',NULL,NULL,10,NULL,100,2,0.2,NULL);
INSERT INTO `sessions` VALUES (2,'2023-05-02 08:00:00.0+02:00','2023-05-02 12:00:00.0+02:00','Carport',NULL,'blauer e-Golf',NULL,NULL,2.5,NULL,88.21,0.75,0.3,NULL);
INSERT INTO `sessions` VALUES (3,'2023-05-03 16:00:00.0+02:00','2023-05-03 20:00:00.0+02:00','Carport',NULL,'weißes Model 3',NULL,NULL,2.5,NULL,50,0.25,0.1,NULL);
INSERT INTO `sessions` VALUES (4,'2023-05-04 22:00:00.0+02:00','2023-05-05 06:00:00.0+02:00','Garage',NULL,'weißes Model 3',NULL,NULL,5,NULL,0,2.5,0.5,NULL);