BIN := eth-custodial
BUILD_CONF := CGO_ENABLED=1 GOOS=linux GOARCH=amd64
BUILD_COMMIT := $(shell git rev-parse --short HEAD 2> /dev/null)
DEBUG := DEV=true

.PHONY: build run clean docs gen-service-token

clean:
	rm ${BIN} ${BOOTSTRAP_BIN}

build:
	${BUILD_CONF} go build -ldflags="-X main.build=${BUILD_COMMIT} -s -w" -o build/${BIN} cmd/service/*.go
	${BUILD_CONF} go build -ldflags="-X main.build=${BUILD_COMMIT} -s -w" -o build/gen-service-token cmd/gen-service-token/main.go

run:
	${BUILD_CONF} ${DEBUG} go run cmd/service/*.go

docs:
	swag fmt --dir internal/api/
	swag init --v3.1 --parseDependency --dir internal/api/ -g swagger.go

gen-service-token:
	${BUILD_CONF} ${DEBUG} go run cmd/gen-service-token/main.go -service localdev