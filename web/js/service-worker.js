console.log("Service-worker loaded.");

// Incrementing OFFLINE_VERSION will kick off the install event and force
// previously cached resources to be updated from the network.
const OFFLINE_VERSION = 1;
const CACHE_NAME = 'treningheten-cache';
// Customize this with a different URL if needed.
const urlsToCache = [
  '/',
	'/json/manifest.json',
	'assets/favicons/favicon.ico'
];

self.addEventListener('install', (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(CACHE_NAME);
    // Setting {cache: 'reload'} in the new request will ensure that the response
    // isn't fulfilled from the HTTP cache; i.e., it will be from the network.
    for(var i = 0; i < urlsToCache.length; i++) {
        await cache.add(new Request(urlsToCache[i], {cache: 'reload'}));
    }
  })());
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Enable navigation preload if it's supported.
    // See https://developers.google.com/web/updates/2017/02/navigation-preload
    if ('navigationPreload' in self.registration) {
      await self.registration.navigationPreload.enable();
    }
  })());

  // Tell the active service worker to take control of the page immediately.
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
    // We only want to call event.respondWith() if this is a navigation request
    // for an HTML page.
    if (event.request.mode === 'navigate') {
        event.respondWith((async () => {
            try {
                // First, try to use the navigation preload response if it's supported.
                const preloadResponse = await event.preloadResponse;
                if (preloadResponse) {
                  return preloadResponse;
                }

                const networkResponse = await fetch(event.request);
                return networkResponse;
            } catch (error) {
                // catch is only triggered if an exception is thrown, which is likely
                // due to a network error.
                // If fetch() returns a valid HTTP response with a response code in
                // the 4xx or 5xx range, the catch() will NOT be called.
                console.log('Fetch failed; returning offline page instead.', error);

                const cache = await caches.open(CACHE_NAME);
                const cachedResponse = await cache.match(OFFLINE_URL);
                return cachedResponse;
            }
          })());
    }

    // If our if() condition is false, then this fetch handler won't intercept the
    // request. If there are any other fetch handlers registered, they will get a
    // chance to call event.respondWith(). If no fetch handlers call
    // event.respondWith(), the request will be handled by the browser as if there
    // were no service worker involvement.
});

self.addEventListener('notificationclose', event => {
    const notification = event.notification;
    const primaryKey = notification.data.primaryKey;

    console.log('Closed notification: ' + primaryKey);
});

self.addEventListener('notificationclick', event => {
    const notification = event.notification;
    const primaryKey = notification.data.primaryKey;
    const url = notification.data.url;
    const action = event.action;

    if (action === 'close') {
        notification.close();
    } else {
        clients.openWindow(url);
        notification.close();
    }

    // TODO 5.3 - close all notifications when one is clicked

});

self.addEventListener('push', event => {

    console.log("Pushing notification.")

    let jsonData;

    if (event.data) {
        jsonData = event.data.json();
    } else {
        console.log("Failed to parse notification data to JSON.")
        jsonData = {
          category: "general",
          title: "Treningheten",
          body: "Treningheten"
        }
    }

    let url;
    let action;

    if(jsonData.category == "achievement") {
      url = "/achievements"
      action = "Check out"
    } else {
      url = "/"
      action = "Visit"
    }
   
    console.log(event.data.json())

    const options = {
        body: jsonData.body,
        icon: 'assets/logo/version4/logo_round_red.png',
        badge: 'assets/logo/version4/logo_round_red_trans.png',
        vibrate: [100, 50, 100],
        data: {
          dateOfArrival: Date.now(),
          primaryKey: 1,
          url: url
        },
        actions: [
            {action: 'explore', title: action,
              icon: 'images/checkmark.png'},
            {action: 'close', title: 'Close',
              icon: 'images/xmark.png'},
        ],
        tag: 'Message'
    };

    event.waitUntil(
        self.registration.showNotification(jsonData.title, options)
    );
});