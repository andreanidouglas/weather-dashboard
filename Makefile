go_files := $(wildcard template/*.go)

all: server view

server: cmd/api/main.go
	go mod tidy
	go tool templ generate -v
	go build -o out/$@ $<

view: view/input.css view/package-lock.json $(go_files)
	cd view && npm run build:assets && npx tailwindcss --input ./input.css --output ./src/css/style.css --minify

clean:
	rm out/*
	rm view/src/css/style.css

docker-build: api.Dockerfile frontend.Dockerfile
	docker build . -f api.Dockerfile -t andreanidouglas/weather-dashboard-api:latest
	docker build . -f frontend.Dockerfile -t andreanidouglas/weather-dashboard-frontend:latest
