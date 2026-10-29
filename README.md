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

The per-location seven-day forecast uses OpenWeatherMap One Call API 3.0. Make sure
the key has access to that API; current-weather cards continue to use the Current
Weather API.

## Dependencies

Make sure you have the following dependencies installed

1. golang>=1.26
2. node>=22
3. templ>=0.3.1070
4. gnu make>=4.3
5. [optional] docker or podman


## Docker

You can use docker to deploy, using the provided `dockerfile` and `compose.yml`

1. first edit the compose.yml file to provide the `API_KEY` environment variable

2. execute the `docker-build` step 

```bash
make docker-build
docker compose up -d
```
