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

1. golang>=1.23
2. node>=v.21.6.2
3. templ>=0.3.819
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

