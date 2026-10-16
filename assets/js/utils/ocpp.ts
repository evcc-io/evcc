import type { Ocpp } from "@/types/evcc";

export function getOcppUrl(ocpp: Ocpp): string {
  // User specified url, e.g., for reverse proxy setups
  if (ocpp.status?.externalUrl) {
    return ocpp.status.externalUrl;
  }
  return `ws://${window.location.hostname}:${ocpp.config.port}/`;
}

export function getOcpp20Url(ocpp: Ocpp): string {
  return `ws://${window.location.hostname}:${ocpp.config.port20}/`;
}

export function getOcppUrlWithStationId(ocpp: Ocpp): string {
  return `${getOcppUrl(ocpp)}<stationId>`;
}

export function getOcpp20UrlWithStationId(ocpp: Ocpp): string {
  return `${getOcpp20Url(ocpp)}<stationId>`;
}
