// evcc PWA Service Worker — handles Web Push notifications

self.addEventListener("push", (event) => {
  if (!event.data) return;

  let data = {};
  try {
    data = event.data.json();
  } catch {
    data = { title: "evcc", body: event.data.text() };
  }

  const title = data.title || "evcc";
  const options = {
    body: data.body || "",
    icon: "/meta/android-chrome-192x192.png",
    badge: "/meta/android-chrome-monochrome.svg",
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

// Handle browser-initiated subscription rotation (endpoint expiry).
self.addEventListener("pushsubscriptionchange", (event) => {
  const options = event.oldSubscription?.options ?? { userVisibleOnly: true };

  event.waitUntil(
    self.registration.pushManager
      .subscribe(options)
      .then((subscription) => {
        const json = subscription.toJSON();
        return fetch("/api/push/subscribe", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            endpoint: json.endpoint,
            auth: json.keys?.auth,
            p256dh: json.keys?.p256dh,
            userAgent: self.navigator?.userAgent ?? "",
          }),
        });
      })
      .catch((err) => {
        console.warn("[push] pushsubscriptionchange resubscribe failed:", err);
      })
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if (client.url.includes(self.location.origin) && "focus" in client) {
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow("/");
      }
    })
  );
});
