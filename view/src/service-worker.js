const SHELL_CACHE = "weather-dashboard-shell-v1";
const WEATHER_CACHE = "weather-dashboard-weather-v1";
const MAX_WEATHER_ENTRIES = 40;

const SHELL_ASSETS = [
    "/",
    "/index.html",
    "/about.html",
    "/offline.html",
    "/manifest.json",
    "/favicon.svg",
    "/favicon.ico",
    "/icons/icon-192.png",
    "/icons/icon-512.png",
    "/icons/icon-maskable-512.png",
    "/icons/apple-touch-icon.png",
    "/css/style.css",
    "/js/htmx.min.js",
    "/js/alpine.min.js",
    "/js/pwa.js",
];

self.addEventListener("install", (event) => {
    event.waitUntil(
        caches.open(SHELL_CACHE)
            .then((cache) => cache.addAll(SHELL_ASSETS))
            .then(() => self.skipWaiting()),
    );
});

self.addEventListener("activate", (event) => {
    event.waitUntil((async () => {
        const currentCaches = new Set([SHELL_CACHE, WEATHER_CACHE]);
        const cacheNames = await caches.keys();
        await Promise.all(cacheNames
            .filter((name) => name.startsWith("weather-dashboard-") && !currentCaches.has(name))
            .map((name) => caches.delete(name)));
        await self.clients.claim();
    })());
});

self.addEventListener("fetch", (event) => {
    const request = event.request;
    if (request.method !== "GET") {
        return;
    }

    const url = new URL(request.url);
    if (url.origin !== self.location.origin) {
        return;
    }

    if (isWeatherCardRequest(url.pathname)) {
        event.respondWith(networkFirstWeather(request));
        return;
    }
    if (request.mode === "navigate") {
        event.respondWith(networkFirstPage(request));
        return;
    }
    event.respondWith(cacheFirstAsset(request));
});

function isWeatherCardRequest(pathname) {
    if (!pathname.startsWith("/api/")) {
        return false;
    }
    return ![
        "/api/cache",
        "/api/suggest",
        "/api/weather",
    ].includes(pathname) && !pathname.startsWith("/api/text/");
}

async function networkFirstPage(request) {
    try {
        const response = await fetch(request);
        if (response.ok && response.type === "basic") {
            const cache = await caches.open(SHELL_CACHE);
            await cache.put(request, response.clone());
        }
        return response;
    } catch {
        const cachedPage = await caches.match(request);
        if (cachedPage) {
            return cachedPage;
        }
        return (await caches.match("/offline.html")) || Response.error();
    }
}

async function cacheFirstAsset(request) {
    const cache = await caches.open(SHELL_CACHE);
    const cachedAsset = await cache.match(request);
    if (cachedAsset) {
        return cachedAsset;
    }

    const response = await fetch(request);
    if (response.ok && response.type === "basic") {
        await cache.put(request, response.clone());
    }
    return response;
}

async function networkFirstWeather(request) {
    const cache = await caches.open(WEATHER_CACHE);
    try {
        const response = await fetch(request);
        if (response.ok && response.type === "basic") {
            await cache.put(request, response.clone());
            await trimWeatherCache(cache);
        }
        if (response.status >= 500) {
            const cachedResponse = await cache.match(request);
            if (cachedResponse) {
                return cachedResponse;
            }
        }
        return response;
    } catch {
        const cachedResponse = await cache.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        return new Response("Weather is unavailable offline until it has been loaded online.", {
            status: 503,
            headers: { "Content-Type": "text/plain; charset=utf-8" },
        });
    }
}

async function trimWeatherCache(cache) {
    const requests = await cache.keys();
    const excess = requests.length - MAX_WEATHER_ENTRIES;
    if (excess <= 0) {
        return;
    }
    await Promise.all(requests.slice(0, excess).map((request) => cache.delete(request)));
}
