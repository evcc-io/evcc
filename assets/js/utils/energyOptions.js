export function optionStep(maxEnergy) {
  if (maxEnergy < 1) return 0.05;
  if (maxEnergy < 2) return 0.1;
  if (maxEnergy < 5) return 0.25;
  if (maxEnergy < 10) return 0.5;
  if (maxEnergy < 25) return 1;
  if (maxEnergy < 50) return 2;
  return 5;
}

export function fmtEnergy(energy, step, fmtKWh, zeroText) {
  if (energy === 0) {
    return zeroText;
  }
  const inKWh = step >= 0.1;
  const digits = inKWh && step < 1 ? 1 : 0;
  return fmtKWh(energy * 1e3, inKWh, true, digits);
}

export function estimatedSoc(energy, socPerKwh) {
  if (!socPerKwh) return null;
  return Math.round(energy * socPerKwh);
}

export function energyOptions(fromEnergy, maxEnergy, socPerKwh, fmtKWh, zeroText) {
  const step = optionStep(maxEnergy);
  const result = [];
  for (let energy = 0; energy <= maxEnergy; energy += step) {
    let text = fmtEnergy(energy, step, fmtKWh, zeroText);
    const disabled = energy < fromEnergy / 1e3 && energy !== 0;
    const soc = estimatedSoc(energy, socPerKwh);
    if (soc) {
      text += ` (+${Math.round(soc)}%)`;
    }
    // prevent rounding errors
    const energyNormal = energy.toFixed(3) * 1;
    result.push({ energy: energyNormal, text, disabled });
  }
  return result;
}
