import type { OcppConfig } from "@/types/evcc";

export function getOcppUrl(ocppConfig: OcppConfig): string {
  // User specified url, e.g., for reverse proxy setups
  if (ocppConfig.externalUrl) {
    return ocppConfig.externalUrl;
  }
  return `ws://${window.location.hostname}:${ocppConfig.port}/`;
}

export function getOcppUrlWithStationId(ocppConfig: OcppConfig): string {
  return `${getOcppUrl(ocppConfig)}<station-id>`;
}
