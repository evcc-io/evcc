// Data source: gCO2eq/kWh, current data is from 2024.
const source = "https://ourworldindata.org/grapher/carbon-intensity-electricity?tab=table";

// This is a manual selection of countries. If yours is missing, please add it with data from the source above.
const regions = [
  { name: "Australia", co2: 552 },
  { name: "Austria", co2: 103 },
  { name: "Belgium", co2: 118 },
  { name: "Canada", co2: 175 },
  { name: "Czech Republic", co2: 414 },
  { name: "Denmark", co2: 143 },
  { name: "Estonia", co2: 341 },
  { name: "Europe", co2: 284 },
  { name: "Finland", co2: 72 },
  { name: "France", co2: 44 },
  { name: "Germany", co2: 344 },
  { name: "Netherlands", co2: 253 },
  { name: "Norway", co2: 31 },
  { name: "Poland", co2: 615 },
  { name: "Sweden", co2: 36 },
  { name: "Switzerland", co2: 37 },
  { name: "United Kingdom", co2: 211 },
  { name: "United States", co2: 384 },
  { name: "World", co2: 473 },
];

export default { regions, source };
