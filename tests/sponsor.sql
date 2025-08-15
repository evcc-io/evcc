BEGIN;

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

-- expired sponsor token
INSERT INTO settings("key", value) VALUES('sponsorToken', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJldmNjLmlvIiwic3ViIjoidHJpYWwiLCJleHAiOjE3NTQ5OTI4MDAsImlhdCI6MTc1MzY5NjgwMCwic3BlIjp0cnVlLCJzcmMiOiJtYSJ9.XKa5DHT-icCM9awcX4eS8feW0J_KIjsx2IxjcRRQOcQ');

-- loadpoint with charger that requires sponsorship
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(3, 1, 'template', '', '', 'Easee Home', '{"charger":"EH123456","password":"none","template":"easee","timeout":"20s","user":"test@example.org"}');
INSERT INTO configs(id, class, type, title, icon, product, value) VALUES(4, 5, '', '', '', '', '{"charger":"db:3","circuit":"","meter":"","phasesConfigured":0,"soc":{"poll":{"mode":"charging","interval":3600000000000},"estimate":true},"thresholds":{"enable":{"delay":60000000000,"threshold":0},"disable":{"delay":180000000000,"threshold":0}},"title":"Carport","vehicle":""}');

COMMIT;