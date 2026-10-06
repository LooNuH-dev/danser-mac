# source this file to get build env for danser on macOS
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
export CGO_ENABLED=1
export CGO_CFLAGS=""
export CGO_LDFLAGS="-L$ROOT/deps/lib -Wl,-rpath,$ROOT/deps/lib"
export PATH="$ROOT/deps/dotnet:$PATH"
export DOTNET_ROOT="$ROOT/deps/dotnet"
