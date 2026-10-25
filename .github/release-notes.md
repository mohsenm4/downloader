## Install

1. Download `Downloader-…-macos-universal.zip` below and unzip it.
2. Move **Downloader.app** to **Applications**.
3. Install the tools it uses:
   ```sh
   brew install yt-dlp ffmpeg whisper-cpp
   ```
4. For speech to text, add a whisper model. See [Whisper model](https://github.com/mohsenm4/downloader#whisper-model) in the README.

### First open

The app is not notarized by Apple, so macOS blocks it the first time.
Open it once, then go to **System Settings → Privacy & Security** and click **Open Anyway**.

Or run this once in Terminal:

```sh
xattr -dr com.apple.quarantine /Applications/Downloader.app
```

## What's new

See [CHANGELOG.md](https://github.com/mohsenm4/downloader/blob/main/CHANGELOG.md).

The zip has one app for both Apple Silicon and Intel Macs. The Intel build is made in the same run but has not been tested on an Intel Mac yet.
