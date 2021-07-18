#
# github.com/vidfamn/OGSGameNotifier
#

VERSION="v0.1-$(shell git rev-parse --short HEAD)"
BUILD_TIME=$(shell date +"%Y%m%d.%H%M%S")

build: clean
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}'" -o bin/OGSGameNotifier-amd64-linux main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}' -H 'windowsgui'" -o bin/OGSGameNotifier-amd64-windows.exe main.go

# GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}'" -o bin/OGSGameNotifier-amd64-darwin main.go

clean:
	-rm -rf bin
