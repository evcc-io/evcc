import api from "../api";

// urlBase64ToUint8Array converts a base64url VAPID public key to a Uint8Array.
function urlBase64ToUint8Array(base64String: string): ArrayBuffer {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = atob(base64);
  const buf = new ArrayBuffer(rawData.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < rawData.length; i++) view[i] = rawData.charCodeAt(i);
  return buf;
}

// registerServiceWorker registers sw.js and subscribes to push notifications.
// Permission is requested once; if the user previously granted it the subscription
// is renewed silently.
export async function registerServiceWorker(): Promise<void> {
  if (!("serviceWorker" in navigator) || !("PushManager" in window) || !("Notification" in window))
    return;

  let reg: ServiceWorkerRegistration;
  try {
    reg = await navigator.serviceWorker.register("/sw.js");
  } catch (e) {
    console.warn("[push] service worker registration failed:", e);
    return;
  }

  // Wait for the service worker to be active.
  await navigator.serviceWorker.ready;

  // Check if already subscribed in the browser.
  const existing = await reg.pushManager.getSubscription();
  if (existing) {
    // Verify the endpoint is still known to the backend.
    // If it was purged (e.g. FCM returned 410), force a fresh subscription.
    try {
      const checkRes = await fetch(
        `/api/push/check?endpoint=${encodeURIComponent(existing.endpoint)}`
      );
      if (checkRes.ok) {
        // Still valid — re-save to keep UserAgent current.
        await saveSubscription(existing);
        return;
      }
      // Not found in backend — unsubscribe from browser to get a fresh endpoint.
      await existing.unsubscribe();
    } catch (e) {
      // Network error during check — keep existing subscription, retry next load.
      console.warn("[push] subscription check failed, keeping existing:", e);
      return;
    }
  }

  // Request notification permission (shows browser prompt if not yet decided).
  const permission = await Notification.requestPermission();
  if (permission !== "granted") return;

  // Fetch VAPID public key and subscribe.
  try {
    const { data } = await api.get("push/vapidkey");
    const publicKey = urlBase64ToUint8Array(data.publicKey);
    const subscription = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: publicKey,
    });
    await saveSubscription(subscription);
  } catch (e) {
    console.warn("[push] subscription failed:", e);
  }
}

// unsubscribeCurrentBrowser removes the push subscription for this browser
// from both the browser's PushManager and the evcc backend.
export async function unsubscribeCurrentBrowser(): Promise<void> {
  if (!("serviceWorker" in navigator)) return;
  const reg = await navigator.serviceWorker.getRegistration("/sw.js");
  if (!reg) return;
  const sub = await reg.pushManager.getSubscription();
  if (!sub) return;
  const endpoint = sub.endpoint;
  try {
    await sub.unsubscribe();
  } catch (e) {
    console.warn("[push] browser unsubscribe failed:", e);
  }
  try {
    await fetch(`/api/push/subscribe?endpoint=${encodeURIComponent(endpoint)}`, {
      method: "DELETE",
    });
  } catch (e) {
    console.warn("[push] backend subscription removal failed:", e);
  }
}

async function saveSubscription(subscription: PushSubscription): Promise<void> {
  const json = subscription.toJSON();
  await api.post("push/subscribe", {
    endpoint: json.endpoint,
    auth: json.keys?.["auth"],
    p256dh: json.keys?.["p256dh"],
    userAgent: navigator.userAgent,
  });
}
