BEGIN;

CREATE TABLE `settings` (
    `key` text
  , `value` text
  , PRIMARY KEY(`key`)
);

INSERT INTO settings("key", value) VALUES('messaging', 'events:
  start:
    title: Charge started
    msg: Started charging in "${mode}" mode
  stop:
    title: Charge finished
    msg: Finished charging ${chargedEnergy:%.1fk}kWh in ${chargeDuration}.
  connect:
    title: Car connected
    msg: "Car connected at ${pvPower:%.1fk}kW PV"
  disconnect:
    title: Car disconnected
    msg: Car disconnected after ${connectedDuration}
  soc:
    title: Soc updated
    msg: Battery charged to ${vehicleSoc:%.0f}%
  guest:
    title: Unknown vehicle
    msg: Unknown vehicle, guest connected?
  asleep:
    title: Vehicle asleep
    msg: Charge release, vehicle {{ if .vehicleTitle }}{{ .vehicleTitle }} {{ end }}not charging.

services:
- type: pushover
  app: pushoverToken
  recipients:
    - recipient1
    - recipient2
    - recipient3
  devices:
    - device1
    - device2
    - device3
- type: telegram
  token: telegramToken
  chats:
    - 12345
    - -54321
    - 111
- type: email
  uri: smtp://john.doe:secret123@emailserver.example.com:587/?fromAddress=john.doe@mail.com&toAddresses=recipient@mail.com
- type: shout
  uri: gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1
- type: ntfy
  uri: https://ntfy.sh/evcc_alert,evcc_pushmessage
  priority: low
  tags: +1,blue_car
- type: custom
  encoding: title
  send:
    cmd: /usr/local/bin/evcc "Title: {{.send}}"
    source: script');

COMMIT;