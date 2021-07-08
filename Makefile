#
# github.com/vidfamn/OGSGameNotifier
#

include .env.dev
export
export VERSION=$(shell git rev-parse --short HEAD)

build: clean
	go build -ldflags="-X 'main.Version=${VERSION}' -X 'main.OAuthClientID=${OAUTH_CLIENT_ID}' -X 'main.OAuthClientSecret=${OAUTH_CLIENT_SECRET}'" -o bin/OGSGameNotifier main.go

clean:
	-rm -rf bin
