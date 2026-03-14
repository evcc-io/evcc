# Automated Fixes

## Issue #28192

**Title:** Tariff solar forcast with mqtt not possible

**URL:** https://github.com/evcc-io/evcc/issues/28192

**Type:** unknown

**Description:**
## Description

I tried to define my solar forecast with a mqtt request (Discussion #28180). The process of proofing hangs and I get no logs (only timeout for port 7070).

The same request worked with the old non GUI tariff and forcast definitions. Tried it with and without interval. Nothing works via GUI definition.


## Steps to Reproduce

Define tariffs and forecasts with new GUI.
Try the following custom solar forecast:

interval: 5m
forecast:
  source: mqtt
  topic: evcc/solcast
  timeout: 

**Fixed at:** 2026-03-14T14:07:25.748310
