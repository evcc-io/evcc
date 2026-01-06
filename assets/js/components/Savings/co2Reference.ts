// Data source: gCO2eq/kWh
const source = "https://ourworldindata.org/grapher/carbon-intensity-electricity?tab=table";

// This is a manual selection of countries. If yours is missing, please add it with data from the source above.
const regions = [
  { name: "Australia", co2: 503 },
  { name: "Austria", co2: 158 },
  { name: "Canada", co2: 128 },
  { name: "Czech Republic", co2: 415 },
  { name: "Denmark", co2: 181 },
  { name: "Estonia", co2: 464 },
  { name: "Europe", co2: 278 },
  { name: "Finland", co2: 131 },
  { name: "France", co2: 85 },
  { name: "Germany", co2: 385 },
  { name: "Netherlands", co2: 356 },
  { name: "Norway", co2: 29 },
  { name: "Poland", co2: 635 },
  { name: "Sweden", co2: 45 },
  { name: "Switzerland", co2: 46 },
  { name: "United Kingdom", co2: 257 },
  { name: "United States", co2: 367 },
  { name: "World", co2: 436 },
];

export default { regions, source };
