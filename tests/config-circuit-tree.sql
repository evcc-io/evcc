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

-- meters
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 2, 'template', '', '', 'Demo meter', '{"template":"demo-meter","usage":"grid","power":"8000","currentL1":"12","currentL2":"12","currentL3":"12"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(2, 2, 'template', '', '', 'Demo meter', '{"template":"demo-meter","usage":"grid","power":"4000","currentL1":"6","currentL2":"6","currentL3":"6"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(3, 2, 'template', '', '', 'Demo meter', '{"template":"demo-meter","usage":"grid","power":"18000","currentL1":"27","currentL2":"27","currentL3":"27"}');

-- chargers
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(4, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"2000","enabled":"true"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(5, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"B","power":"0","enabled":"false"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(6, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"1400","enabled":"true"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(7, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"1000","enabled":"true"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(17, 1, 'template', '', '', 'Demo heat pump', '{"template":"demo-heatpump","operationMode":"heating","power":"2500","enabled":"true"}');

-- circuits: Home (grid meter) > Garage (dedicated meter) > Corner (no meter); Home > Carport (no meter); Home > Workshop (dedicated meter, over limit)
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(8, 6, 'template', 'Home', '', 'Static circuit', '{"template":"static","parent":"","meter":"db:1","maxpower":20000,"maxcurrent":32}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(9, 6, 'template', 'Garage', '', 'Static circuit', '{"template":"static","parent":"db:8","meter":"db:2","maxpower":11000,"maxcurrent":16}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(10, 6, 'template', 'Corner', '', 'Static circuit', '{"template":"static","parent":"db:9","maxcurrent":16}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(11, 6, 'template', 'Carport', '', 'Static circuit', '{"template":"static","parent":"db:8","maxpower":5000}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(12, 6, 'template', 'Workshop in the basement', '', 'Static circuit', '{"template":"static","parent":"db:8","meter":"db:3","maxcurrent":25}');

-- loadpoints
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(13, 5, '', '', '', '', '{"title":"Wallbox left","charger":"db:4","circuit":"db:9","phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(14, 5, '', '', '', '', '{"title":"Wallbox right","charger":"db:5","circuit":"db:9","phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(15, 5, '', '', '', '', '{"title":"Motorbike","charger":"db:6","circuit":"db:10","mode":"now","phasesConfigured":1,"minCurrent":6,"maxCurrent":6,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(16, 5, '', '', '', '', '{"title":"Carport","charger":"db:7","circuit":"db:11","phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');

INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(18, 5, '', '', '', '', '{"title":"Heat pump","charger":"db:17","circuit":"db:12","phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');

INSERT INTO settings("key", value) VALUES('gridMeter', 'db:1');

COMMIT;
