# Optimizer

Optimizer uses mixed integer linear programming (MILP) to minize a cost function. The optimizer model implementation honors:

- energy consumption/ feed-in costs
- solar forecast
- base load (aka home) energy demand
- end of forecast commercial value
- strategy- either "charge before export" (charge loads as soon as possible) or attenuating grid peaks on the demand side, the feed-in side or both
- home battery or loadpoint/vehicle...
  - capacity, soc and charge goals
  - charge/discharge power limits and efficiency

Optimization spans N slots with N being minimum of available forecast data.

## Parameters

### Energy consumption/ feed-in costs/ Solar forecast

Grid/ feed-in/ solar tariff data.

TODO

- [ ] make feed-in optional
- [ ] make solar optional

### Base load energy demand

Collected 15min energy profile over the last 4 weeks, averaged per slot. A per-slot percentile (e.g. 50 = median, robust against a few heavy days) can be configured via `/api/profilepercentile` instead.

### Forecast uncertainty

By default the optimizer plans on the forecasts as they are. A risk profile makes it hedge against forecast errors: `POST /api/optimizerriskprofile` with `{"solar": 0.3, "demand": 0.2, "budget": 8}` assumes that a slot may deliver 30% less solar and need 20% more energy than forecasted, in at most 8 slots per day. `DELETE /api/optimizerriskprofile` removes it, `GET /api/optimizerriskprofile` returns it.

### Measured value blending

The solar forecast and the base load profile are anchored to the current situation
using the last completed 15min metrics slot, decaying linearly over 4 slots:

- base load: the measured home consumption replaces the first slot and decays into the profile
- solar: the scale factor measured production/forecasted production is applied to the first slot and decays towards 1

### End of forecast commercial value

Use minimum of energy consumption cost.

### Home Battery

### Loadpoint and Vehicles

- home battery or loadpoint/vehicle...
  - capacity, soc and charge goals
  - charge/discharge power limits and efficiency

Without vehicle capacity or soc a configured session energy limit is modelled instead:
state is the session's charged energy, goal is the limit. Loadpoints with neither are
not modelled at all- their power is added to the base load.
