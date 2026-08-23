GO ?= go
GOGIO ?= gogio

CORE_PKGS := ./internal/core/... ./internal/platform/... ./internal/terminal/... ./internal/session/...

.PHONY: tidy fmt test test-core vet build linux android check-android clean

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./cmd/... ./internal/...

test:
	$(GO) test ./...

test-core:
	CGO_ENABLED=0 $(GO) test $(CORE_PKGS)

vet:
	$(GO) vet ./...

build: fmt test vet
	$(GO) build ./cmd/ishgo

linux: build
	mkdir -p bin
	$(GO) build -o bin/ishgo ./cmd/ishgo

check-android:
	@command -v $(GOGIO) >/dev/null 2>&1 || (echo "error: gogio is required; install gioui.org/cmd/gogio" >&2; exit 1)
	@test -n "$(ANDROID_HOME)$(ANDROID_SDK_ROOT)" || (echo "error: ANDROID_HOME or ANDROID_SDK_ROOT is required" >&2; exit 1)

android: check-android
	mkdir -p bin
	$(GOGIO) -target android -appid com.mostafa637.ishgo -o bin/ishgo.apk ./cmd/ishgo

clean:
	rm -rf bin
