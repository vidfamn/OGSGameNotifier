#
# github.com/vidfamn/OGSGameNotifier
#

VERSION="v0.1.1-$(shell git rev-parse --short HEAD)"
BUILD_TIME=$(shell date +"%Y%m%d.%H%M%S")

build: clean
	mkdir -p bin/assets
	cp assets/notification_icon.png bin/assets

	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}'" -o bin/OGSGameNotifier-amd64-linux .
	# GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}' -H 'windowsgui'" -o bin/OGSGameNotifier-amd64-windows.exe main.go

# GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.Build=${BUILD_TIME}'" -o bin/OGSGameNotifier-amd64-darwin main.go

release: clean build
	tar -czvf bin/OGSGameNotifier-amd64-linux-${VERSION}.tar.gz bin/OGSGameNotifier-amd64-linux bin/assets
	tar -czvf bin/OGSGameNotifier-amd64-windows-${VERSION}.tar.gz bin/OGSGameNotifier-amd64-windows.exe bin/assets

clean:
	-rm -rf bin
