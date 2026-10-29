# Weather Dashboard


A simple http page, that will return cards with weather for selected cities

## Build

Use the provided `Makefile` to build all necessary components

```bash
make
```

## Frontend integration tests

The forecast modal integration test runs the dashboard and Alpine.js in Happy DOM
and mocks the forecast API response; it does not need an OpenWeatherMap key.

```bash
cd view
npm ci
npm test
```

## Running

The API server requires the `API_KEY` environment variable and serves API routes
only. The static frontend is served separately by the frontend container.

`API_KEY` should be a valid OpenWeatherMap.org api key

The per-location forecast uses OpenWeatherMap's [Daily Forecast 16 Days API](https://openweathermap.org/api/forecast16?collection=current_forecast)
and displays the next seven full days. The API key must have access to that
forecast product; current-weather cards continue to use the Current Weather API.

## Dependencies

Make sure you have the following dependencies installed

1. Go 1.27.3 or newer
2. node>=22
3. templ v0.3.1100 (managed by `go tool`)
4. gnu make>=4.3
5. [optional] docker or podman

## Progressive Web App

The frontend includes a web app manifest and a service worker that caches the
application shell and up to 40 recently loaded weather cards for offline use.
Weather refreshes, city suggestions, and first-time weather lookups still need
a network connection. For installation, serve the site on HTTPS (or `localhost`)
and visit it once while online so the service worker can cache the app shell.


## Docker

You can use docker to deploy, using the provided `dockerfile` and `compose.yml`

1. first edit the compose.yml file to provide the `API_KEY` environment variable

2. execute the `docker-build` step 

```bash
make docker-build
docker compose up -d
```
