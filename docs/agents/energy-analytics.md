# Energy Analytics

How the Energy page turns 15-minute meter slots into attribution, cost, CO2, autarky and forecast accuracy. This is the conceptual model and the reasoning behind it. The code in `core/metrics/` and `assets/js/views/Energy.vue` is the reference for details.

## Data model

Everything derives from two tables written every 15 minutes:

- **Meter slots**: per entity `energy` and `returnEnergy` in kWh, optionally SoC or temperature. Entities belong to a group: `pv`, `grid`, `battery`, `home`, `loadpoint`, `consumer`, `meter`, `forecast`, `temperature`.
- **Tariff slots**: grid price, feed-in price, CO2 intensity and outside temperature at the same slot boundaries. Each value is optional, a slot exists as soon as one is known.

Directions per group:

| group     | energy      | returnEnergy |
| --------- | ----------- | ------------ |
| grid      | import      | export       |
| battery   | charge      | discharge    |
| pv        | production  | unused       |
| home      | consumption | unused       |
| loadpoint | charged     | unused       |
| consumer  | consumed    | unused       |
| meter     | metered     | reverse      |

`home` is not a meter. It is the residual of the live power balance: grid + pv + battery discharge minus loadpoint charging, clamped at zero, integrated per slot. It is the whole house load excluding loadpoints. Consumers are sub-meters of home; `meter` entities are monitoring only and never enter any balance.

Everything works on slots, never on session records. Sessions use an instantaneous solar share, the slot model nets import and export inside 15 minutes. The two will disagree on "solar %", by design. The Energy page follows the slot model, Sessions stay as they are.

## Source attribution

Each slot has sources (PV production, battery discharge, grid import) and sinks (home, battery charge, loadpoints, export). Which source fed which sink is not measurable, so it is assigned by a priority rule, applied per slot and summed over the period:

1. Sources in order PV, battery, grid.
2. Sinks in order home, battery charge, loadpoints, export.
3. Walk sources outer, sinks inner, moving `min(remaining source, remaining sink)` each step. Battery does not feed itself, grid does not feed export.

This is the same rule the live control loop uses for the green share, so the page and the running system tell the same story. Home comes before loadpoints because the house load is not steerable while charging is. Anything a slot's sources cannot cover (meter mismatch, coarse counters, missing entity) is silently dropped rather than invented. Attributed totals are therefore always at or below the metered totals.

Everything downstream is built on these flows: the flow chart, the source mix bars (share of PV, battery, grid in what reached a sink), autarky, cost and CO2.

The overview bar chart uses a simpler per-bucket split on the client: production covers export first, then battery charging, the remainder is self-consumption. The usage view keeps that column, from minus import and discharge up to production, and fills it with sinks instead: export on top, then consumers, each loadpoint, each battery at the bottom, each with all it received from production and import alike. The baseline just cuts through the stack. Energy no sink explains stays with consumption, a sink no source covers is not drawn.

## Metered versus attributed

Headline numbers that a user can compare against their meter (grid import, grid export, production, home consumption, battery charge and discharge) are metered sums. Numbers that describe relationships (autarky, self-consumption, source mix, savings) are derived from the attribution. The two can differ slightly, which is accepted. A cost box never claims to explain more energy than the tariff-covered attributed energy.

## Derived ratios

- **Autarky** = 1 - grid share of what reached home and loadpoints.
- **Self-consumption** = 1 - export share of PV production.
- **Source mix per sink** = share of each source in the energy that reached that sink.

All ratios are clamped to 0 to 100 percent and fall back to zero when the denominator is zero.

## Cost and CO2

Prices are applied at slot level and summed, never as period average times energy. A day of cheap solar noon plus expensive evening import must not be priced at the daily mean.

Per slot with a grid price:

- Import cost = import × grid price. Export revenue = export × feed-in price.
- **Consumption cost**: grid energy that reached home or loadpoints at the grid price, plus self-produced energy that reached them at the feed-in price. The feed-in price is the opportunity cost: that kWh could have been sold.
- Each flow into home or loadpoints carries its own cost and priced energy under the same rule. Consumption cost is the sum of these flows, and it can be split by source.
- **Baseline**: the same consumed energy at the period's average grid price, as if there were no PV and no battery. The page derives it from the average grid price and the priced consumption.
- **Savings** = baseline - consumption cost.

CO2 works the same way with grid intensity, but self-produced energy counts as zero, so consumption CO2 is only the grid share.

Slots without a price (or CO2 value) are excluded from the cost model on both sides. Every cost is therefore reported together with the energy it covers. The displayed "priced energy" is what the savings figure covers, so a partially priced period is visibly smaller instead of wrong. An average price is always cost divided by priced energy, never by all energy: the month in which price recording began would otherwise show a price that is far too low. The effective price per kWh shown next to savings is consumption cost divided by priced consumption, compared with the average grid price.

The grid history carries the same pair per bucket: import cost and feed-in revenue with their priced energy. The grid chart uses them for the amount and the effective price of a day or month.

Per-entity cost uses average sink rates, not slot detail: every consumer is priced at the average rate of home, every loadpoint at the average rate of the non-home consumption. The source breakdown of an entity works the same way: it inherits the source shares and prices of its sink, scaled to its energy. Sub-meters are not attributed individually because their slot pattern would have to be matched against the home slot pattern with no gain in trust.

## Forecast accuracy

The PV forecast is persisted per slot alongside the meters. Accuracy for a finished period:

- Per bucket (slot, day or month depending on the view), error = |forecast - produced|.
- Accuracy = 1 - Σ error / Σ forecast.

Absolute error per bucket is used so that an over-forecast morning and an under-forecast afternoon do not cancel out. The total delta (forecast minus produced) is shown separately as "x kWh above/below" so the user sees both the shape error and the level bias. In the chart the same data is drawn as floating segments from the forecast value to the produced value.

For the running day the accuracy is not yet meaningful, so the tile shows the remaining expected production instead.

Buckets that have no persisted forecast slot (restart gaps, forecast provider outage) are skipped entirely: they carry no error, no anchor and are not drawn. Showing a full-height bar for "forecast unknown, production known" would read as a giant miss.

The forecast source changed over time in real installations. Comparing accuracy across periods with different providers is not supported, and there is no "valid since" marker for forecast quality.

## Missing, partial and implausible data

Principles:

- **Hide, do not fake.** A card whose group has no metered energy in the period is not rendered. The page's empty state keys on any metered energy, not on attribution, so a PV-only or battery-only setup still gets a page.
- **Show what is covered.** Cost and CO2 tiles print the priced energy they explain. Without any tariff data they show a "no price data" state with a link to tariff configuration instead of zeros.
- **Drop, do not stretch.** Attribution never scales a source up to meet a sink. Unexplained energy disappears from ratios rather than being assigned to grid by default.
- **Clamp residuals at zero.** "Others" (home minus tracked consumers) is clamped per slot so meter noise or a consumer briefly exceeding home never produces negative consumption.
- **Slot gaps are tolerated in sums.** A day with 91 to 95 slots after a restart sums fine. Recovered slots (downtime energy written into one slot) are kept in sums but excluded from profile averages, where they would distort the shape.

Known data quality traits that are accepted, not fixed:

- Chargers that report energy hourly (Easee) lump an hour into one slot. Daily sums are right, slot-level solar share and cost for those loadpoints are approximate.
- Battery counters with coarse resolution can show cumulative discharge above charge. No efficiency figure is derived from them. Throughput and SoC are fine.
- Entities have different "valid since" dates. Autarky without a grid meter, or savings before tariffs were persisted, is simply absent or covers less energy.

## Tariff aggregation for charts

The tariff read path can aggregate slots per hour, day or month, returning the mean plus min and max per bucket. These aggregates are display context only. Cost figures always come from the slot-level model above.

The grid chart shows prices in a panel below the bars. The day view draws the raw 15-minute tariff. Month and year views draw the effective price per bucket, which is the bucket's cost divided by its priced energy. A plain per-day tariff average would be a constant for a fixed two-zone tariff and hide when energy was actually bought cheap or expensive. A bucket without energy falls back to the tariff average so the line stays continuous.

A price is only charted when it changes over the period. A constant price still appears in the legend and the tooltip. The legend shows the tariff range for a day and the average price paid for longer periods.

## Deciding new metrics

Before adding a number to the page:

1. Is it a metered sum or an attribution result? Label it so it does not get compared against the wrong thing.
2. What does it show when the underlying entity or tariff is absent? Hide or state coverage. No zeros that look like measurements.
3. Does it cancel errors (net delta) where per-bucket errors matter (absolute)? Pick deliberately.
4. Can slot-level pricing express it? If it needs average price times energy, it belongs in a subline, not a headline.
