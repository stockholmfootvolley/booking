PACKAGE = github.com/stockholmfootvolley/booking

build: bin/booking
bin/booking:
	GOOS=linux GOARCH=amd64 go build -o bin/booking cmd/main.go