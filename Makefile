#
# github.com/vidfamn/OGSGameNotifier
#

VERSION=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date +"%Y%m%d.%H%M%S")

build: clean
	go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}'" -o bin/OGSGameNotifier main.go

clean:
	-rm -rf bin
