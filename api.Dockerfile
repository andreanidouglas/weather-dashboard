FROM golang:1.23.1-bookworm AS build
WORKDIR /app
ENV CGO_ENABLED=0
COPY . .
RUN go build -o server cmd/api/main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /app/server /app/server
EXPOSE 8000
ENTRYPOINT [ "/app/server" ]
