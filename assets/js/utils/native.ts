export function isApp() {
  return navigator.userAgent.includes("evcc/");
}

export function appDetection() {
  if (isApp()) {
    const $html = document.querySelector("html");
    $html?.classList.add("app");
  }
}

export function hasAppCapability(capability: string): boolean {
  return isApp() && window.evccAppCapabilities?.includes(capability) === true;
}

type AppMessage =
  | { type: "online" | "offline" | "settings" }
  | { type: "download"; url: string; method?: string; body?: unknown };

export function sendToApp(data: AppMessage) {
  window.ReactNativeWebView?.postMessage(JSON.stringify(data));
}

export function handleDownloadClick(event: Event, url: string) {
  if (dispatchDownload(url)) {
    event.preventDefault();
  }
}

export function dispatchDownload(url: string, method?: string, body?: unknown): boolean {
  if (!hasAppCapability("download")) return false;
  const absolute = new URL(url, window.location.href).toString();
  sendToApp({ type: "download", url: absolute, method, body });
  return true;
}
