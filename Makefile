BINARY=argocd-helm-ext-plugin

default: build

build:
	go build -ldflags="-s -w" -o release/${BINARY} .

install: build
