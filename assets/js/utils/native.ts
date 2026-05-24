export function isApp() {
  return navigator.userAgent.includes("evcc/");
}

export function appDetection() {
  if (isApp()) {
    const $html = document.querySelector("html");
    $html?.classList.add("app");
  }
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

// Intercept `<a download>` clicks when running inside the native app — see
// appDownloadFile() for the underlying mechanism. Outside the app this is a
// no-op and the browser handles the link normally.
export function appDownloadHandler(url: string) {
  return (event: Event) => {
    if (appDownloadFile(url)) {
      event.preventDefault();
    }
  };
}

// Hand a download to the native app over the message bridge. The app runs
// the fetch inside its webview (so server cookies, basic auth, and any
// future auth scheme reach the download) and presents the result via the
// system share sheet. `init` lets callers issue POST/etc. downloads with a
// body — used by BackupRestoreModal so the app can fetch the auth-cookie-
// protected /system/backup endpoint without the password ever leaving the
// webview. Returns true when the app handled it; outside the app it returns
// false so callers can fall back to the browser-native download path.
export function appDownloadFile(url: string, init?: AppDownloadInit): boolean {
  if (!isApp()) return false;
  const absolute = new URL(url, window.location.href).toString();
  sendToApp({ type: "download", url: absolute, ...init });
  return true;
}
