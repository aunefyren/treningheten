console.log("Service-worker loaded.");

// Incrementing OFFLINE_VERSION will kick off the install event and force
// previously cached resources to be updated from the network.
const OFFLINE_VERSION = 2;
const CACHE_NAME = 'treningheten-cache-v' + OFFLINE_VERSION;
const OFFLINE_URL = '/offline';
const urlsToCache = [
    '/',
    '/offline',
    '/manifest.json',
    '/service-worker.js',
    '/robots.txt',
    'assets/favicons/favicon.ico',
    'assets/refresh-cw.svg'
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

        // Delete old caches that don't match the current CACHE_NAME.
        const cacheNames = await caches.keys();
        await Promise.all(
            cacheNames
                .filter(name => name !== CACHE_NAME)
                .map(name => {
                    console.log('Deleting old cache:', name);
                    return caches.delete(name);
                })
        );
    })());

    // Tell the active service worker to take control of the page immediately.
    self.clients.claim();
});

self.addEventListener('fetch', (event) => {
    // Navigation requests (HTML pages): network-first, fall back to offline page.
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
                console.log('Fetch failed; returning offline page instead.', error);
                const cache = await caches.open(CACHE_NAME);
                const cachedResponse = await cache.match(OFFLINE_URL);
                return cachedResponse;
            }
        })());
        return;
    }

    // Static assets (scripts, styles, images): cache-first, fall back to network.
    const dest = event.request.destination;
    if (dest === 'script' || dest === 'style' || dest === 'image' || dest === 'font') {
        event.respondWith((async () => {
            const cache = await caches.open(CACHE_NAME);
            const cachedResponse = await cache.match(event.request);
            if (cachedResponse) {
                return cachedResponse;
            }
            try {
                const networkResponse = await fetch(event.request);
                // Cache successful responses for future offline use.
                if (networkResponse.ok) {
                    cache.put(event.request, networkResponse.clone());
                }
                return networkResponse;
            } catch (error) {
                console.log('Asset fetch failed:', event.request.url, error);
                // Nothing we can do for assets — just fail silently.
                return new Response('', { status: 408, statusText: 'Offline' });
            }
        })());
    }
});

self.addEventListener('notificationclose', event => {

    try {

        const notification = event.notification;
        const primaryKey = notification.data.primaryKey;

        console.log('Closed notification: ' + primaryKey);

    } catch(e) {
        console.log("Failed to click notification. Error: " + e)
    }
});

self.addEventListener('notificationclick', event => {

    try {
        
        const notification = event.notification;
        const primaryKey = notification.data.primaryKey;
        const url = notification.data.url;
        const action = event.action;

        if (action === 'close') {
            notification.close();
        } else {
            const promiseChain = clients.openWindow(url);
            event.waitUntil(promiseChain);
            notification.close();
        }

        console.log('Clicked notification: ' + primaryKey);
    
    } catch(e) {
        console.log("Failed to click notification. Error: " + e)
    }

    // TODO 5.3 - close all notifications when one is clicked

});




self.addEventListener('push', function(event) {
    if (!(self.Notification && self.Notification.permission === "granted")) {
        console.log("Notification permission not given.")
        return;
    }
    
    console.log("Pushing notification.")
    
    try {

        let jsonData = event.data?.json() ?? {
            category: "general",
            title: "Error",
            body: "An error occurred"
        };

        console.log("JSON: " + JSON.stringify(jsonData));

        let url;
        let action;

        if(jsonData.category == "achievement") {
            url = "/achievements"
            action = "Check out"
        } else if(jsonData.category == "news") {
            url = "/news"
            action = "Read"
        } else if(jsonData.category == "debt") {
            url = "/wheel?debt_id=" + jsonData.additional_data
            action = "Spin"
        } else {
            url = "/"
            action = "Visit"
        }

        const options = {
            body: jsonData.body,
            icon: '/assets/logos/logo-384x384.png',
            badge: '/assets/logos/logo-mono-96x96.png',
            vibrate: [100, 50, 100],
            data: {
                dateOfArrival: Date.now(),
                primaryKey: 1,
                url: url
            },
            actions: [
                {action: 'explore', title: action,
                    icon: '/assets/check.svg'
                },
                {action: 'close', title: 'Close',
                    icon: '/assets/x.svg'
                },
            ],
            tag: 'Message'
        };

        event.waitUntil(
            self.registration.showNotification(jsonData.title, options)
        );

    } catch(e) {
        console.log("Failed to push notification. Error: " + e)
    }
});
