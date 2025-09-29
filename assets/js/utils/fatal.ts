import type { FatalError } from "@/types/evcc";

const FATALS = ["configfile", "database"];

function isError(fatal: FatalError[]) {
  return fatal.length > 0;
}

export function isUserConfigError(fatal: FatalError[]) {
  if (!isError(fatal)) {
    return false;
  }

  if (fatal.some((f) => FATALS.includes(f.class ?? ""))) {
    return false;
  }

  return true;
}

export function isSystemError(fatal: FatalError[]) {
  return isError(fatal) && !isUserConfigError(fatal);
}
