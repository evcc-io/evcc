export function isApp() {
  return navigator.userAgent.includes("evcc/");
}

export function appDetection() {
  if (isApp()) {
    const $html = document.querySelector("html");
    $html?.classList.add("app");
  }
}

type AppMessage = { type: "online" | "offline" | "settings" } | { type: "download"; url: string };

export function sendToApp(data: AppMessage) {
  window.ReactNativeWebView?.postMessage(JSON.stringify(data));
}

// Intercept `<a download>` clicks when running inside the native app and hand
// the URL to the app via the message bridge. The app then performs the HTTP
// request natively (so server cookies, basic auth, and any future auth scheme
// reach the download) and presents the result via the system share sheet.
// Outside the app this is a no-op and the browser handles the link normally.
export function appDownloadHandler(url: string) {
  return (event: Event) => {
    if (!isApp()) return;
    event.preventDefault();
    const absolute = new URL(url, window.location.href).toString();
    sendToApp({ type: "download", url: absolute });
  };
}
