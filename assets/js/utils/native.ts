export function isApp() {
  return navigator.userAgent.includes("evcc/");
}

export function appDetection() {
  if (isApp()) {
    const $html = document.querySelector("html");
    $html?.classList.add("app");
  }
}

export function sendToApp(data: { type: string }) {
  window.ReactNativeWebView?.postMessage(JSON.stringify(data));
}
