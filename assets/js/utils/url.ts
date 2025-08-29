// Get the base URL for API endpoints including protocol, host, and port
export function getBaseUrl(): string {
  const { protocol, hostname, port } = window.location;
  // Include port only if it's not the default for the protocol
  const portPart = port && port !== "80" && port !== "443" ? `:${port}` : "";
  return `${protocol}//${hostname}${portPart}`;
}
