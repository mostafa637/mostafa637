GO ?= go
GOGIO ?= gogio

.PHONY: tidy fmt test vet build linux android clean

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./cmd/... ./internal/...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build: fmt test vet
	$(GO) build ./cmd/ishgo

linux: build
	$(GO) build -o bin/ishgo ./cmd/ishgo

android:
	$(GOGIO) -target android -appid com.mostafa637.ishgo -o bin/ishgo.apk ./cmd/ishgo

clean:
	rm -rf bin
