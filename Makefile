.PHONY: run test race vet build app shots tools

run:
	go run ./cmd/app

test:
	go test ./...

race:
	go test -race ./...

vet:
	gofmt -l .
	go vet ./...

# plain binary in ./bin
build:
	go build -o bin/downloader ./cmd/app

# macOS app bundle: cmd/app/Downloader.app
app:
	cd cmd/app && fyne package -os darwin

# draw the window to PNG files (no need to open the app)
shots:
	UI_SHOTS=$(CURDIR)/docs/screenshots go test -count=1 -run Screenshots ./internal/interface/ui

# the programs the app runs
tools:
	brew install yt-dlp ffmpeg whisper-cpp
	go install fyne.io/tools/cmd/fyne@latest
