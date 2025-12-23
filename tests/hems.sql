DROP TABLE IF EXISTS `grid_sessions`;
CREATE TABLE `grid_sessions` (
  `id` integer,
  `created` datetime,
  `finished` datetime,
  `type` text,
  `grid_power` real,
  `limit_power` real,
  PRIMARY KEY (`id`)
);

INSERT INTO `grid_sessions` VALUES (1,'2025-05-01 08:00:00.0+02:00','2025-05-01 09:30:00.0+02:00','consumption',5000,4200);
INSERT INTO `grid_sessions` VALUES (2,'2025-05-02 14:00:00.0+02:00','2025-05-02 15:00:00.0+02:00','feedin',3500,3000);
INSERT INTO `grid_sessions` VALUES (3,'2025-05-05 10:00:00.0+02:00','2025-05-05 11:00:00.0+02:00','consumption',4800,4200);
