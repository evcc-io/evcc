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

type AppDownloadInit = {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
};

type AppMessage =
  | { type: "online" | "offline" | "settings" }
  | ({ type: "download"; url: string } & AppDownloadInit);

export function sendToApp(data: AppMessage) {
  window.ReactNativeWebView?.postMessage(JSON.stringify(data));
}

export function appDownloadHandler(url: string) {
  return (event: Event) => {
    if (appDownloadFile(url)) {
      event.preventDefault();
    }
  };
}

export function appDownloadFile(url: string, init?: AppDownloadInit): boolean {
  if (!hasAppCapability("download")) return false;
  const absolute = new URL(url, window.location.href).toString();
  sendToApp({ type: "download", url: absolute, ...init });
  return true;
}
