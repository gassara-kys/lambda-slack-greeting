APPNAME := $(shell basename `pwd`)
SRC := $(shell ls *.go | grep -v '_test.go')

.PHONY: all
all: run

.PHONY: tidy
tidy: $(SRC)
	go mod tidy

.PHONY: fmt
fmt: tidy
	go fmt

.PHONY: test
test: fmt
	go test -v -cover .

.PHONY: install
install:
	go install github.com/okzk/go-lambda-runner

.PHONY: callback
callback: test install
	source env.sh && \
	cat lambda_callback.json | go-lambda-runner go run $(SRC)

.PHONY: url
url: test install
	source env.sh && \
	cat lambda_url_valification.json | go-lambda-runner go run $(SRC)

.PHONY: run
run: test install
	source env.sh && \
	cat lambda_callback.json | go-lambda-runner go run $(SRC)

.PHONY: build
build: test
	$(shell GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o bin/$(APPNAME) -v $(SRC) )

.PHONY: zip
zip: build
	cd bin && zip $(APPNAME).zip $(APPNAME)
