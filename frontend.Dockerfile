FROM node:lts-bookworm-slim AS builder
COPY ./view/package.json \
    ./view/package-lock.json \
    ./view/tailwind.config.js \
    ./view/input.css \
    /app/

WORKDIR /app
RUN npm ci

COPY ./view/src/index.html /app/src/index.html
COPY ./view/src/js/ /app/src/js/
RUN npx tailwindcss -i input.css -o src/css/style.css
RUN ls /app/

FROM rtsp/lighttpd:latest
COPY --from=builder /app/src/ /var/www/html/
