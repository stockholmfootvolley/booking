PACKAGE = github.com/stockholmfootvolley/booking
SHORT_HASH=$(shell git rev-parse --short HEAD)
build: bin/booking
bin/booking:
	GOOS=linux GOARCH=amd64 go build -o bin/booking cmd/main.go

docker:
	rm bin/booking
	docker build . -t booking:$(SHORT_HASH)
