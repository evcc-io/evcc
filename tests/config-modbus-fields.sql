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

-- using RFC 5737 TEST-NET addresses (192.0.2.0/24, 198.51.100.0/24) that are guaranteed to fail connection attempts
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 2, 'template', 'TCP Test', '', 'SunSpec Inverter', '{"host":"192.0.2.1","id":10,"modbus":"tcpip","port":5020,"template":"sunspec-inverter","usage":"pv"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(2, 2, 'template', 'RTU/IP Test', '', 'SunSpec Inverter', '{"host":"198.51.100.1","id":20,"modbus":"rs485tcpip","port":8899,"template":"sunspec-inverter","usage":"pv"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(3, 2, 'template', 'Serial Test', '', 'SunSpec Inverter', '{"baudrate":19200,"comset":"8E1","device":"/dev/ttyUSB5","id":30,"modbus":"rs485serial","template":"sunspec-inverter","usage":"pv"}');

INSERT INTO settings("key", value) VALUES('pvMeters', 'db:1,db:2,db:3');

COMMIT;
