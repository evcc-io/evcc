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

INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 2, 'template', 'TCP Test', '', 'SunSpec Inverter', '{"host":"192.168.1.10","id":10,"modbus":"tcpip","port":5020,"template":"sunspec-inverter","usage":"pv"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(2, 2, 'template', 'RTU/IP Test', '', 'SunSpec Inverter', '{"host":"192.168.1.20","id":20,"modbus":"rs485tcpip","port":8899,"template":"sunspec-inverter","usage":"pv"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(3, 2, 'template', 'Serial Test', '', 'SunSpec Inverter', '{"baudrate":19200,"comset":"8E1","device":"/dev/ttyUSB5","id":30,"modbus":"rs485serial","template":"sunspec-inverter","usage":"pv"}');

INSERT INTO settings("key", value) VALUES('pvMeters', 'db:1,db:2,db:3');

COMMIT;
