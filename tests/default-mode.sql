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

-- loadpoints: charger (class 1) + loadpoint (class 5); "defaultMode" = configured default, "mode" = last mode persisted at runtime

-- lp 0: default fast, last off
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(1, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"0"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(2, 5, '', '', '', '', '{"charger":"db:1","title":"Default fast","defaultMode":"now","mode":"off"}');

-- lp 1: keep as is, last fast
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(3, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"0"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(4, 5, '', '', '', '', '{"charger":"db:3","title":"Keep as is","defaultMode":"","mode":"now"}');

-- lp 2: heater, default fast, last off
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(5, 1, 'template', '', '', 'Demo heat pump', '{"template":"demo-heatpump","operationMode":"heating","power":"0"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(6, 5, '', '', '', '', '{"charger":"db:5","title":"Heater","defaultMode":"now","mode":"off"}');

-- lp 3: pre-0.316 minpv default, always charge not persisted yet
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(7, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"C","power":"0"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(8, 5, '', '', '', '', '{"charger":"db:7","title":"Legacy minpv","defaultMode":"minpv","mode":"now"}');

-- pv meter, required for smart mode and always charge in the UI
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(9, 2, 'template', '', '', 'Demo meter', '{"template":"demo-meter","usage":"pv","power":"0"}');
INSERT INTO settings("key", value) VALUES('pvMeters', 'db:9');

COMMIT;
