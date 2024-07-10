const FATALS = ["configfile", "database"];

function isError(fatal) {
  return !!fatal?.error;
}

export function isUserConfigError(fatal) {
  if (!isError(fatal)) {
    return false;
  }

  const errorClass = fatal?.class;
  if (FATALS.includes(errorClass)) {
    return false;
  }
  return true;
}

export function isSystemError(fatal) {
  return isError(fatal) && !isUserConfigError(fatal);
}
