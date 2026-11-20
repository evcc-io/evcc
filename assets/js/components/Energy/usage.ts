export interface UsageSlot {
  pv: number;
  gridIn: number;
  gridOut: number;
  batOut: number; // discharge, all batteries
  home: number;
  bats: number[]; // charge per battery
  lps: number[]; // charged per loadpoint
}

// what each sink received in the slot
export interface UsageRow {
  export: number;
  bats: number[];
  home: number;
  lps: number[];
}

// Assigns every slot's sources, production first, then battery discharge and grid
// import, to the sinks: export and battery charge from production as in the
// overview, then consumers before loadpoints like the flow attribution. Energy no
// sink claims stays with consumption so the sinks always add up to the sources; a
// sink no source covers is not drawn. Chargers that book their energy hourly show
// up this way, that is a data quality issue the chart does not paper over.
export function usageSplit(slots: UsageSlot[]): UsageRow[] {
  return slots.map((s) => {
    let up = s.pv;
    const fromPv = (v: number) => {
      const x = Math.min(up, Math.max(0, v));
      up -= x;
      return x;
    };
    const exportUp = fromPv(s.gridOut);
    const batUp = s.bats.map(fromPv);
    const homeUp = fromPv(s.home);
    const lpUp = s.lps.map(fromPv);

    let down = s.gridIn + s.batOut;
    const fromImport = (v: number) => {
      const x = Math.min(down, Math.max(0, v));
      down -= x;
      return x;
    };
    const homeDown = fromImport(s.home - homeUp);
    const batDown = s.bats.map((b, i) => fromImport(b - batUp[i]!));
    const lpDown = s.lps.map((l, i) => fromImport(l - lpUp[i]!));

    return {
      export: exportUp,
      bats: batUp.map((u, i) => u + batDown[i]!),
      home: homeUp + up + homeDown + down,
      lps: lpUp.map((u, i) => u + lpDown[i]!),
    };
  });
}
