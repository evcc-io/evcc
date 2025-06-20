import type { FatalError } from "@/types/evcc";

const FATALS = ["configfile", "database"];

function isError(fatal?: FatalError) {
  return !!fatal?.error;
}

export function isUserConfigError(fatal?: FatalError) {
  if (!isError(fatal)) {
    return false;
  }

  const errorClass = fatal?.class;
  if (FATALS.includes(errorClass)) {
    return false;
  }
  return true;
}

export function isSystemError(fatal?: FatalError) {
  return isError(fatal) && !isUserConfigError(fatal);
}
