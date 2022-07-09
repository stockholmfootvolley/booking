PACKAGE = github.com/stockholmfootvolley/booking
COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BRANCH := `git rev-parse --abbrev-ref HEAD 2>/dev/null`
VERSION := `git describe --abbrev=0 --tags 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`
GO_SRC = $(shell git ls-files *.go */**.go go.mod go.sum)


build: bin/booking
bin/booking: $(GO_SRC)
	GOOS=linux GOARCH=amd64 go build -o bin/booking