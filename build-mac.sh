#!/bin/bash
# Builds universal (arm64 + x86_64) danser for macOS into ./dist/danser and ./dist/danser.app
set -e
cd "$(dirname "$0")"
source ./env.sh

VERSION=${1:-dev}
OUT=./dist/danser
mkdir -p "$OUT"

export CGO_LDFLAGS="-L$ROOT/deps/lib -Wl,-rpath,@executable_path"

(cd danser && go run tools/assets/assets.go ./ "../$OUT/")

LDFLAGS="-s -w -X 'github.com/wieku/danser-go/build.VERSION=$VERSION' -X 'github.com/wieku/danser-go/build.Stream=Release' -X 'github.com/wieku/danser-go/build.DanserExec=danser'"
for GA in arm64 amd64; do
  [ $GA = amd64 ] && CA=x86_64 || CA=arm64
  (cd danser && GOOS=darwin GOARCH=$GA CC="clang -arch $CA" CGO_CFLAGS="-mmacosx-version-min=12.0" \
    CGO_LDFLAGS="$CGO_LDFLAGS -mmacosx-version-min=12.0" go build -trimpath -ldflags "$LDFLAGS" \
    -tags "exclude_cimgui_glfw exclude_cimgui_sdli" -o "../$OUT/danser-$GA" .)
done
lipo -create "$OUT/danser-arm64" "$OUT/danser-amd64" -output "$OUT/danser"
rm "$OUT/danser-arm64" "$OUT/danser-amd64"

cp deps/lib/*.dylib "$OUT/"
mkdir -p "$OUT/ffmpeg"
cp deps/ffmpeg/ffmpeg deps/ffmpeg/ffprobe "$OUT/ffmpeg/"

# self-contained so end users don't need .NET installed; single-file bundles can't be lipo'd,
# so ship both and pick the right one with a tiny wrapper
rm -rf "$OUT/lazer-bridge"
for RID in osx-arm64 osx-x64; do
  dotnet publish lazer-bridge/LazerBridge.csproj -c Release -r $RID --self-contained \
    -p:PublishSingleFile=true -p:IncludeNativeLibrariesForSelfExtract=true -p:DebugType=none \
    -o "$OUT/lazer-bridge/$RID" | grep -E "error|->" || true
done
cat > "$OUT/lazer-bridge/lazer-bridge" <<'SH'
#!/bin/sh
D="$(cd "$(dirname "$0")" && pwd)"
[ "$(sysctl -n hw.optional.arm64 2>/dev/null)" = 1 ] && R=osx-arm64 || R=osx-x64
exec "$D/$R/lazer-bridge" "$@"
SH
chmod +x "$OUT/lazer-bridge/lazer-bridge"

# ---- danser.app bundle (data goes to ~/Library/Application Support/danser)
APP=./dist/danser.app
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp -R "$OUT/danser" "$OUT/assets.dpak" "$OUT"/*.dylib "$OUT/ffmpeg" "$OUT/lazer-bridge" "$APP/Contents/MacOS/"

ICONSET=$(mktemp -d)/danser.iconset
mkdir -p "$ICONSET"
for sz in 16 32 128 256 512; do
  sips -z $sz $sz danser/assets/textures/coinbig.png --out "$ICONSET/icon_${sz}x${sz}.png" >/dev/null
  sips -z $((sz*2)) $((sz*2)) danser/assets/textures/coinbig.png --out "$ICONSET/icon_${sz}x${sz}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/danser.icns"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>danser</string>
  <key>CFBundleDisplayName</key><string>danser</string>
  <key>CFBundleIdentifier</key><string>com.github.wieku.danser</string>
  <key>CFBundleExecutable</key><string>danser</string>
  <key>CFBundleIconFile</key><string>danser</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>LSMinimumSystemVersion</key><string>12.0</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSApplicationCategoryType</key><string>public.app-category.games</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key><string>osu! replay</string>
      <key>CFBundleTypeExtensions</key><array><string>osr</string></array>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSHandlerRank</key><string>Alternate</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key><string>osu! beatmap archive</string>
      <key>CFBundleTypeExtensions</key><array><string>osz</string></array>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSHandlerRank</key><string>Alternate</string>
    </dict>
  </array>
</dict>
</plist>
PLIST

# ad-hoc signature so Gatekeeper on Apple Silicon accepts the bundle locally
codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || true

# zip for releases
(cd dist && rm -f danser-macos-universal.zip && ditto -c -k --keepParent danser.app danser-macos-universal.zip)

echo "Built into $OUT and $APP (dist/danser-macos-universal.zip)"
