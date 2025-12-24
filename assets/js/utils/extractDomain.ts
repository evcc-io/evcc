export const extractDomain = (url: string): string => {
  const urlObj = new URL(url);
  let hostname = urlObj.hostname;

  // ipv6
  if (hostname.startsWith("[") && hostname.endsWith("]")) {
    hostname = hostname.slice(1, -1);
  }

  // ipv4
  if (/^(\d{1,3}\.){3}\d{1,3}$/.test(hostname)) {
    return hostname;
  }

  // domain
  return hostname.split(".").slice(-2).join(".");
};
