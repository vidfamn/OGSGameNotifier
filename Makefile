#
# github.com/vidfamn/OGSGameNotifier
#

include .env.dev
export
export VERSION=$(shell git rev-parse --short HEAD)

build: clean
	go build -ldflags="-X 'main.Version=${VERSION}'" -o bin/OGSGameNotifier main.go

clean:
	-rm -rf bin
