# Chart Conventions

Design rules for charts in the UI (history, sessions, forecast, optimizer).

## Power and energy y-axes

The axis adapts to the size of the data so tick labels always read as distinct, round numbers:

1. When the peak stays below 1 kW(h), the axis shows W/Wh. Without this, every tick of a small system (e.g. balcony PV) rounds to "0 kW". Charts without data also show the W scale.
2. In W mode the axis always spans at least 1000 W(h), so small values keep a stable visual context instead of filling the full chart height.
3. Ticks are whole numbers. Only when the axis tops out between 1 and 3 kW(h) do ticks get one decimal, because whole numbers would repeat (0 / 0.5 / 1 instead of 0 / 1 / 1).
4. The axis range ends on a round number just above the peak (1, 2, 3, 4, 6 or 8 times a power of ten).

The same stepping applies to other kilo-scaled units, e.g. g/kg for CO2.

Presentation:

- Tick labels are plain numbers without a unit suffix.
- The unit appears once as a small label above the axis ("W", "kW", "Wh", "kWh") where the layout has headroom. Forecast charts omit the label and rely on the tooltip for units.
- Bidirectional charts (charge and discharge, import and export) keep their automatic range and adopt only unit and decimals.

## Tooltips

- One shared look for all charts: a heading with the time or category, rows with name and value, no color swatches.
- Tooltip values carry their unit and pick it per value (auto W/kW), independent of the axis unit.

## Layout

- Value axes sit on the right for bar and history charts.
- Axis units stay short (kg, g/kWh, W).
- Entities sort alphabetically so colors and order stay stable across periods.
- Polar charts round only outer corners; ticks render under the segments.
- Bar charts do not animate.
