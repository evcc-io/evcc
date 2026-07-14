import type { Router } from "vue-router";

let router: Router;

export function isApp() {
  return navigator.userAgent.includes("evcc/");
}

type FromAppMessage = {
  type: "navigate";
  path: string; // e.g. "/forecast", "/?lp=1"
};

function receiveFromApp(message: FromAppMessage) {
  switch (message.type) {
    case "navigate":
      router.push(message.path);
  }
}

export function appDetection(r: Router) {
  router = r;
  if (!isApp()) return;
  document.querySelector("html")?.classList.add("app");

  const handler = (event: Event) => {
    try {
      receiveFromApp(JSON.parse((event as MessageEvent).data));
    } catch {
      return;
    }
  };
  window.addEventListener("message", handler); // iOS
  document.addEventListener("message", handler); // Android
}

export function hasAppCapability(capability: string): boolean {
  return isApp() && window.evccAppCapabilities?.includes(capability) === true;
}

type ToAppMessage =
  | { type: "online" | "offline" | "settings" }
  | { type: "download"; url: string; headers?: Record<string, string> };

export function sendToApp(data: ToAppMessage) {
  window.ReactNativeWebView?.postMessage(JSON.stringify(data));
}

export function handleDownloadClick(event: Event, url: string) {
  if (dispatchDownload(url)) {
    event.preventDefault();
  }
}

export function dispatchDownload(url: string, headers?: Record<string, string>): boolean {
  if (!hasAppCapability("download")) return false;
  const absolute = new URL(url, window.location.href).toString();
  sendToApp({ type: "download", url: absolute, headers });
  return true;
}
