FROM golang:1.23.1-bookworm AS build
RUN go install github.com/a-h/templ/cmd/templ@latest
WORKDIR /app
ENV CGO_ENABLED=0
COPY . .
RUN templ generate
RUN go build -o server cmd/api/main.go

FROM debian:bookworm-slim
LABEL org.opencontainers.image.source=https://github.com/andreanidouglas/weather-dashboard
LABEL org.opencontainers.image.description="Weather Dashboard API Image"
LABEL org.opencontainers.image.licenses=MIT
WORKDIR /app
COPY --from=build /app/server /app/server
EXPOSE 8000
ENV STANDALONE=false
ENTRYPOINT [ "/app/server" ]
