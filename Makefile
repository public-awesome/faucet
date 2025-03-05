build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/faucet-linux cmd/*.go

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/faucet-linux cmd/*.go

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o bin/faucet-mac cmd/*.go

build:
	go build -o bin/faucet-server cmd/*.go

build-docker: build-linux
	docker buildx build --platform linux/amd64 --push -t publicawesome/faucet:latest .

build-docker-arm64: build-linux-arm64
	docker buildx build --platform linux/arm64 --push -t publicawesome/faucet:latest-arm64 .
