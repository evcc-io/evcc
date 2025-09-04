import formatter, { POWER_UNIT } from "../mixins/formatter";

export function optionStep(maxEnergy: number) {
  if (maxEnergy < 0.1) return 0.005;
  if (maxEnergy < 1) return 0.05;
  if (maxEnergy < 2) return 0.1;
  if (maxEnergy < 5) return 0.25;
  if (maxEnergy < 10) return 0.5;
  if (maxEnergy < 25) return 1;
  if (maxEnergy < 50) return 2;
  return 5;
}

export function fmtEnergy(
  energy: number = 0,
  step: number,
  fmtWh: InstanceType<typeof formatter>["fmtWh"],
  zeroText: any
) {
  if (energy === 0) {
    return zeroText;
  }
  const inKWh = step >= 0.1;
  const digits = inKWh && step < 1 ? 1 : 0;
  return fmtWh(energy * 1e3, inKWh ? POWER_UNIT.KW : POWER_UNIT.W, true, digits);
}

export function estimatedSoc(energy: number, socPerKwh?: number) {
  if (!socPerKwh) return null;
  return Math.round(energy * socPerKwh);
}

export function energyOptions(
  fromEnergy: number,
  maxEnergy: number,
  fmtWh: InstanceType<typeof formatter>["fmtWh"],
  fmtPercentage: InstanceType<typeof formatter>["fmtPercentage"],
  zeroText: string,
  socPerKwh?: number,
  selectedValue?: number
) {
  const step = optionStep(maxEnergy);
  const result = [];

  // helper to create option
  const makeOption = (energy: number) => {
    let text = fmtEnergy(energy, step, fmtWh, zeroText);
    const disabled = energy < fromEnergy / 1e3 && energy !== 0;
    const soc = estimatedSoc(energy, socPerKwh);
    if (soc) {
      text += ` (+${fmtPercentage(soc)})`;
    }
    // prevent rounding errors
    const energyNormal = parseFloat(energy.toFixed(3));
    return { energy: energyNormal, text, disabled };
  };

  // add standard increments
  for (let energy = 0; energy <= maxEnergy; energy += step) {
    result.push(makeOption(energy));
  }

  // add selected value if it's not in the list
  if (selectedValue && !result.find((o) => o.energy === selectedValue)) {
    result.push(makeOption(selectedValue));
    result.sort((a, b) => a.energy - b.energy);
  }

  return result;
}
