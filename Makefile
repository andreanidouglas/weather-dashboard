go_files := $(wildcard template/*.go)

all: server view

server: cmd/api/main.go
	go mod tidy
	templ generate
	go build -o out/$@ $<

view: view/input.css $(go_files)
	cd view && npx tailwindcss --input ./input.css --output ./src/css/style.css --minify

clean:
	rm out/*
	rm view/src/css/style.css

docker-build: api.Dockerfile frontend.Dockerfile
	docker build . -f api.Dockerfile -t ghcr.io/andreanidouglas/weather-dashboard:api_latest
	docker build . -f frontend.Dockerfile -t ghcr.io/andreanidouglas/weather-dashboard:frontend_latest

docker-push: docker-build
	docker push ghcr.io/andreanidouglas/weather-dashboard:api_latest
	docker push ghcr.io/andreanidouglas/weather-dashboard:frontend_latest
