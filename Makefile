.PHONY: run test race vet build app release shots tools

VERSION ?= 1.0.0
BUILD   ?= 1
APP     := cmd/app/Downloader.app
ZIP     := Downloader-$(VERSION)-macos-universal.zip

run:
	go run ./cmd/app

test:
	go test ./...

race:
	go test -race ./...

vet:
	test -z "$$(gofmt -l .)"
	go vet ./...

# plain binary in ./bin
build:
	go build -o bin/downloader ./cmd/app

# macOS app bundle for this Mac: cmd/app/Downloader.app
app:
	cd cmd/app && fyne package --os darwin --app-version $(VERSION) --app-build $(BUILD)

# universal app (Apple Silicon + Intel), zipped for a GitHub release, in ./dist
release:
	rm -rf dist $(APP)
	mkdir -p dist
	cd cmd/app && fyne package --os darwin --release --app-version $(VERSION) --app-build $(BUILD)
	CGO_ENABLED=1 GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/app-arm64 ./cmd/app
	CGO_ENABLED=1 GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/app-amd64 ./cmd/app
	lipo -create -output $(APP)/Contents/MacOS/app dist/app-arm64 dist/app-amd64
	rm dist/app-arm64 dist/app-amd64
	codesign --force --deep --sign - $(APP)
	ditto -c -k --keepParent $(APP) dist/$(ZIP)
	cd dist && shasum -a 256 $(ZIP) > $(ZIP).sha256

# draw the window to PNG files (no need to open the app)
shots:
	UI_SHOTS=$(CURDIR)/docs/screenshots go test -count=1 -run Screenshots ./internal/interface/ui

# the programs the app runs
tools:
	brew install yt-dlp ffmpeg whisper-cpp
	go install fyne.io/tools/cmd/fyne@latest
