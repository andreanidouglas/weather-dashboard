# Weather Dashboard


A simple http page, that will return cards with weather for selected cities

## Build

Use the provided `Makefile` to build all necessary components

```bash
make
```

## Running

To run the program standalone, you need to setup the `STANDALONE` and `API_KEY` environment variables.

`API_KEY` should be a valid OpenWeatherMap.org api key

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
