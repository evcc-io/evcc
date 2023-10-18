// Data source: gCO2eq/kWh
const co2 = {
  url: "https://ourworldindata.org/grapher/carbon-intensity-electricity?tab=table",
  year: "2020",
};

// Data source: price/kWh
const price = {
  url: "https://www.globalpetrolprices.com/electricity_prices/",
  year: "2023",
};

// This is a manual selection of countries. If yours is missing, please add it with data from the sources above.
const regions = [
  { name: "Australia", co2: 503, price: 0.36, currency: "AUD" },
  { name: "Austria", co2: 158, price: 0.45, currency: "EUR" },
  { name: "Canada", co2: 128, price: 0.18, currency: "CAD" },
  { name: "Czech Republic", co2: 415, price: 0.36, currency: "EUR" },
  { name: "Denmark", co2: 181, price: 2.4, currency: "DKK" },
  { name: "Estonia", co2: 464, price: 0.29, currency: "EUR" },
  { name: "Europe", co2: 278, price: 0.25, currency: "EUR" }, // price: https://www.statista.com/statistics/1046505/household-electricity-prices-european-union-eu28-country/
  { name: "Finland", co2: 131, price: 0.22, currency: "EUR" },
  { name: "France", co2: 85, price: 0.2, currency: "EUR" },
  { name: "Germany", co2: 385, price: 0.39, currency: "EUR" },
  { name: "Netherlands", co2: 356, price: 0.34, currency: "EUR" },
  { name: "Norway", co2: 29, price: 1.56, currency: "NOK" },
  { name: "Poland", co2: 635, price: 0.21, currency: "EUR" },
  { name: "Sweden", co2: 45, price: 3.16, currency: "SEK" },
  { name: "Switzerland", co2: 46, price: 0.3, currency: "CHF" },
  { name: "United Kingdom", co2: 257, price: 0.19, currency: "GBP" },
  { name: "United States", co2: 367, price: 0.18, currency: "USD" },
];

export default {
  regions,
  sources: { co2, price },
};
