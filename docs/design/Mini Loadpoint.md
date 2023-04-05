# Proposal: Mini Loadpoint

## Motivation

Our loadpoint UI is optimized to visualize charger- and vehicle status. We notice that evcc is more and more used for other energy optimizations like heating (electric heater, heatpump) or smart plugs (battery charging, appliances). Due to evccs plugin structure its already possible to make this work right now, but applying the concept of vehicles to everything feels clunky. Introducing the concept of a "Mini Loadpoint" should address this.

## Concept

The idea is to establish a new type of loadpoint that omits everything thats vehicle-, SoC-sepecific. The concept of a charging session and connection-states will only be available in the "normal" loadpoint.

**Modes**

Mini loadpoints also offer manual control (off/on) as well as an (optional?) evcc controlled mode (pv/smart).

**Read only**

The mini loadpoint can also be used for simple meters without power control to monitor consumption of devices like washers, dryers, simple-heaters, coffee-makers, air conditioners, ... UI controls will be disabled/omitted for these read-only loadpoints.

**Configuration**

Creating/configuring a mini loadpoint would be similar to the existing one since the functionality is mostly a subset of the regular loadpoint.

**Visual representation**

Mini loadpoints will show along regular loadpoints in the UI. Since they have fewer features we can visalize them more compact. On larger screens the space required for a regular loadpoint may be enough for four mini loadpoints. On phone screens we could show two mini loadpoints instead of one regular loadpoint (see screenshots).

## Features: Loadpoint vs Mini Loadpoint

|                                                   | Regular | Mini  |
| ------------------------------------------------- | ------- | ----- |
| Loadpoint title                                   | ✅      | ✅    |
| Loadpoint icon                                    | ❌      | ✅    |
| Loadpoint settings<br>(max power, phases, ...)    | ✅      | ✅    |
| Modes<br>(off, pv/smart, on)                      | ✅      | ✅    |
| Read only<br>(no switching, just watching)        | ❌      | ✅    |
| Status text<br>(connected, timer, ..)             | ✅      | ❌    |
| Simplified status<br>(icon?)                      | ❌      | ✅    |
| Vehicle title                                     | ✅      | ❌    |
| Vehicle icon                                      | ✅      | ❌    |
| Vehicle switcher                                  | ✅      | ❌    |
| Charging plan                                     | ✅      | ❓    |
| SoC / energy limit                                | ✅      | ❌    |
| Progress bar / slider                             | ✅      | ❌    |
| Power                                             | ✅      | ✅    |
| Charged energy                                    | ✅      | 🔁    |
| SoC                                               | ✅      | ❌    |
| Elapsed time                                      | ✅ ℹ️   | ❌    |
| Remaining time                                    | ✅ ℹ️   | ❌    |
| Additional infos<br>(% solar, co2, price/kWh, °C) | 🔮 🔁   | 🔮 🔁 |
| Charging sessions                                 | ✅      | ❌    |
| Cumulative stats<br>(daily, monthly, ...)         | 🔮      | 🔮    |

✅ Available.<br>
❌ Not available.<br>
🔁 User configurable. Maybe toggling through available values.<br>
ℹ️ Currently "remaining" replaces "elapsed" if available (hard wired).<br>
🔮 In the future.
