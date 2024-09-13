all: tidy server view

server: cmd/api/main.go
	go mod tidy
	go build -o out/$@ $<

view: view/input.css
	cd view && npx tailwindcss -i ./input.css -o ./src/css/style.css

clean:
	rm out/*
	rm view/src/css/style.css

docker-build: api.Dockerfile frontend.Dockerfile
	docker build . -f api.Dockerfile -t andreanidouglas/weather-dashboard-api:latest
	docker build . -f frontend.Dockerfile -t andreanidouglas/weather-dashboard-frontend:latest
