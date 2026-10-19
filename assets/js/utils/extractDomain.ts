// returns the full hostname, e.g. "electrify.hesotec.de" stays as is
export const extractHostname = (url: string): string => {
  const urlObj = new URL(url);
  let hostname = urlObj.hostname;

  // ipv6
  if (hostname.startsWith("[") && hostname.endsWith("]")) {
    hostname = hostname.slice(1, -1);
  }

  return hostname;
};

// returns the registrable domain, stripping any subdomain, e.g. "login.example.org" -> "example.org"
export const extractDomain = (url: string): string => {
  const hostname = extractHostname(url);

  // ipv4
  if (/^(\d{1,3}\.){3}\d{1,3}$/.test(hostname)) {
    return hostname;
  }

  // domain
  return hostname.split(".").slice(-2).join(".");
};
