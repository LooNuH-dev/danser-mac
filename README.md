# danser-mac

macOS port of [danser-go](https://github.com/Wieku/danser-go) by Wieku. Universal build, runs natively on Apple Silicon and Intel Macs (macOS 12+).

All the actual danser work is Wieku's and the danser-go contributors'. This repo just makes it build and run on a Mac. If something breaks only on macOS, open an issue here, not upstream.

## What's different from upstream

- Builds as a regular `danser.app`, one binary for arm64 and x86_64
- osu!lazer import: the bundled `lazer-bridge` reads the lazer library and links songs, skins and replays into danser. .NET is bundled, nothing to install
- ffmpeg is included, so recording to mp4 works out of the box
- `.osr` and `.osz` files can be opened from Finder
- Settings and data live in `~/Library/Application Support/danser`

## Install

Grab `danser-macos-universal.zip` from [Releases](https://github.com/LooNuH-dev/danser-mac/releases), unzip it and move `danser.app` to Applications.

The app isn't notarized, so macOS will refuse to open it the first time. Either right-click it and choose Open, or clear the quarantine flag:

```bash
xattr -dr com.apple.quarantine /Applications/danser.app
```

To use it from the terminal:

```bash
/Applications/danser.app/Contents/MacOS/danser -h
```

Command line options are the same as upstream, see [danser/README.md](danser/README.md).

## Building

You need Xcode Command Line Tools, Go, cmake, git and curl. Everything else (SDL3, BASS, libyuv, ffmpeg, .NET SDK) is fetched into `./deps`, nothing is installed system-wide.

```bash
./setup-deps.sh
```

```bash
./build-mac.sh 0.12.0-mac
```

Output goes to `dist/danser.app` and `dist/danser-macos-universal.zip`.

Layout:

- `danser/` - danser-go source with the macOS changes
- `lazer-bridge/` - small C# tool that reads osu!lazer's realm database
- `setup-deps.sh` - builds/downloads native dependencies
- `build-mac.sh` - builds the universal binary and the app bundle

## Known issues

- Some prebuilt dependencies target macOS 15 on Intel, so older Intel Macs may not work.

## License

Same as upstream, see [danser/LICENSE](danser/LICENSE) and [danser/CREDITS.md](danser/CREDITS.md).
