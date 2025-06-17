FROM golang:1.24.4-bookworm AS build
RUN go install github.com/a-h/templ/cmd/templ@latest
WORKDIR /app
ENV CGO_ENABLED=0
COPY . .
RUN templ generate -v
RUN go build -o server cmd/api/main.go

FROM debian:bookworm-slim
LABEL org.opencontainers.image.source=https://github.com/andreanidouglas/weather-dashboard
LABEL org.opencontainers.image.description="Weather Dashboard API Image"
LABEL org.opencontainers.image.licenses=MIT
WORKDIR /app
ENV TZ="America/Sao Paulo"
RUN apt update && apt install ca-certificates curl -y --no-install-recommends
COPY --from=build /app/server /app/server
EXPOSE 8080
ENV STANDALONE=false
ENTRYPOINT [ "/app/server" ]
