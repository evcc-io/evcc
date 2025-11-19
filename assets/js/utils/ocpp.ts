import type { OcppConfig } from "@/types/evcc";

/**
 * Get the OCPP server URL
 * @param ocppConfig - OCPP configuration object
 * @returns The OCPP server URL or null if port is 0
 */
export function getOcppUrl(ocppConfig: OcppConfig): string | null {
	// User specified url, e.g., for reverse proxy setups
	if (ocppConfig.externalUrl) {
		return ocppConfig.externalUrl;
	}

	const port = ocppConfig.port;
	if (!port) return null;

	return `ws://${window.location.hostname}:${port}/`;
}

/**
 * Get the OCPP server URL with station ID placeholder
 * @param ocppConfig - OCPP configuration object
 * @returns The OCPP server URL with <station-id> placeholder or null if not available
 */
export function getOcppUrlWithStationId(ocppConfig: OcppConfig): string | null {
	const url = getOcppUrl(ocppConfig);
	if (url) {
		return `${url}<station-id>`;
	}
	return null;
}
