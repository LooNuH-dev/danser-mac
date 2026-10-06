#!/bin/bash
# Fetches/builds all native dependencies locally into ./deps (nothing is installed system-wide).
# Requires: git, cmake, clang (Xcode CLT), curl, go
# All native libs are built universal (arm64 + x86_64).
set -e
cd "$(dirname "$0")"
mkdir -p deps/dl deps/lib deps/src deps/ffmpeg
cd deps
ARCHS="arm64;x86_64"

# BASS (un4seen) - universal dylibs
for f in bass24-osx bass_fx24-osx bassmix24-osx; do
  [ -f dl/$f.zip ] || curl -fsSL -o dl/$f.zip https://www.un4seen.com/files/$f.zip || curl -fsSL -o dl/$f.zip https://www.un4seen.com/files/z/0/$f.zip
  unzip -oq dl/$f.zip -d dl/$f
done
cp dl/*/libbass*.dylib lib/

# SDL3 (go-sdl3 needs >= 3.4)
[ -d src/SDL ] || git clone -q --depth 1 --branch release-3.4.18 https://github.com/libsdl-org/SDL.git src/SDL
cmake -S src/SDL -B src/SDL/build -DCMAKE_BUILD_TYPE=Release -DCMAKE_OSX_DEPLOYMENT_TARGET=12.0 -DCMAKE_OSX_ARCHITECTURES="$ARCHS" -DSDL_TEST_LIBRARY=OFF >/dev/null
cmake --build src/SDL/build -j"$(sysctl -n hw.ncpu)" >/dev/null
cp -L src/SDL/build/libSDL3.0.dylib lib/libSDL3.dylib

# libyuv (without libjpeg to avoid external deps)
[ -d src/libyuv ] || git clone -q --depth 1 https://chromium.googlesource.com/libyuv/libyuv src/libyuv
# libyuv picks SIMD sources per CMAKE_SYSTEM_PROCESSOR, so build each arch separately and lipo
for A in arm64 x86_64; do
  cmake -S src/libyuv -B src/libyuv/build-$A -DCMAKE_BUILD_TYPE=Release -DCMAKE_OSX_DEPLOYMENT_TARGET=12.0 \
    -DCMAKE_OSX_ARCHITECTURES=$A -DCMAKE_SYSTEM_NAME=Darwin -DCMAKE_SYSTEM_PROCESSOR=$A -DCMAKE_DISABLE_FIND_PACKAGE_JPEG=ON >/dev/null
  cmake --build src/libyuv/build-$A -j"$(sysctl -n hw.ncpu)" --target yuv_shared >/dev/null
done
lipo -create src/libyuv/build-arm64/libyuv.dylib src/libyuv/build-x86_64/libyuv.dylib -output lib/libyuv.dylib

for f in lib/*.dylib; do install_name_tool -id "@rpath/$(basename "$f")" "$f"; done

# static ffmpeg/ffprobe builds for macOS, merged into universal binaries
for FARCH in arm64 amd64; do
  for t in ffmpeg ffprobe; do
    [ -f dl/$t-$FARCH.zip ] || curl -fsSL -o dl/$t-$FARCH.zip "https://ffmpeg.martin-riedl.de/redirect/latest/macos/$FARCH/release/$t.zip"
    mkdir -p dl/ffmpeg-$FARCH && unzip -oq dl/$t-$FARCH.zip -d dl/ffmpeg-$FARCH
  done
done
for t in ffmpeg ffprobe; do
  lipo -create dl/ffmpeg-arm64/$t dl/ffmpeg-amd64/$t -output ffmpeg/$t
done

# local .NET SDK for lazer-bridge
curl -fsSL -o dl/dotnet-install.sh https://dot.net/v1/dotnet-install.sh
bash dl/dotnet-install.sh --channel 10.0 --install-dir ./dotnet >/dev/null

echo "Dependencies ready in ./deps"
