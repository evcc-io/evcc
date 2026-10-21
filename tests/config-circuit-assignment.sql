BEGIN;

CREATE TABLE configs (
    id integer PRIMARY KEY AUTOINCREMENT,
    class integer,
    type text,
    title text,
    icon text,
    product text,
    value text
);

CREATE TABLE settings (
    key text,
    value text,
    PRIMARY KEY(key)
);

-- chargers
INSERT INTO configs VALUES(4, 1, 'template', '', '', 'Demo charger', '{"template":"demo-charger","status":"B","power":"0","enabled":"false"}');
INSERT INTO configs VALUES(5, 1, 'template', '', '', 'Demo heat pump', '{"template":"demo-heatpump","operationMode":"heating","power":"0","enabled":"false"}');

-- circuits: Home > Garage
INSERT INTO configs VALUES(8, 6, 'template', 'Home', '', 'Static circuit', '{"template":"static","parent":"","maxpower":20000,"maxcurrent":32}');
INSERT INTO configs VALUES(9, 6, 'template', 'Garage', '', 'Static circuit', '{"template":"static","parent":"db:8","maxpower":11000,"maxcurrent":16}');

-- loadpoints
INSERT INTO configs VALUES(13, 5, '', '', '', '', '{"title":"Wallbox left","charger":"db:4","circuit":"","minSoc":35,"priority":4,"solarShare":0.6,"phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');
INSERT INTO configs VALUES(14, 5, '', '', '', '', '{"title":"Heat pump","charger":"db:5","circuit":"db:8","phasesConfigured":3,"minCurrent":6,"maxCurrent":16,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}}}');

COMMIT;
