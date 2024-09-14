FROM node:lts-bookworm-slim AS builder
COPY . /app/

WORKDIR /app/view
RUN npm ci
RUN npx tailwindcss -i input.css -o src/css/style.css
RUN ls /app/

FROM rtsp/lighttpd:latest
COPY --from=builder /app/view/src/ /var/www/html/
