export function isApp() {
  return navigator.userAgent.includes("evcc-app");
}

export function appDetection() {
  if (isApp()) {
    const $html = document.querySelector("html");
    $html.classList.add("app");
  }
}
