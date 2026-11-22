BEGIN;

CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

INSERT INTO settings("key", value) VALUES('circuits', '- name: main
  maxcurrent: 16
  maxPower: 10000
- name: child
  parent: main
  maxcurrent: 10');

COMMIT;
