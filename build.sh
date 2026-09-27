#!/usr/bin/env bash
# Build every platform for which the matching official OfficeCLI exists in tools/.
set -euo pipefail

root_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
tools_dir="$root_dir/tools"
release_dir="$root_dir/release"
mkdir -p "$release_dir"

if [[ "${OFFICECLI_PATCH_SKIP_FETCH:-0}" != "1" ]]; then
  python_bin="${PYTHON:-python3}"
  if ! command -v "$python_bin" >/dev/null 2>&1; then
    python_bin="python"
  fi
  "$python_bin" "$root_dir/scripts/fetch_officecli.py" --tools-dir "$tools_dir" \
    --version "${OFFICECLI_PATCH_OFFICECLI_VERSION:-latest}"
fi

# Keep Go's temporary build cache inside the project so Git Bash does not need to
# write to a protected Windows user-cache location.
export CGO_ENABLED=0
export GOCACHE="$root_dir/.gocache"

build_target() {
  local goos="$1"
  local goarch="$2"
  local official_name="$3"
  local output_name="$4"
  local tags="${5:-}"
  local official_path="$tools_dir/$official_name"

  if [[ ! -f "$official_path" ]]; then
    printf '略過 %s：tools/%s 尚未放入。\n' "$output_name" "$official_name" >&2
    return
  fi

  printf '建置 %s\n' "$output_name"
  if [[ -n "$tags" ]]; then
    GOOS="$goos" GOARCH="$goarch" go build -buildvcs=false -trimpath \
      -tags "$tags" -ldflags='-s -w' -o "$release_dir/$output_name" .
  else
    GOOS="$goos" GOARCH="$goarch" go build -buildvcs=false -trimpath \
      -ldflags='-s -w' -o "$release_dir/$output_name" .
  fi
}

cd "$root_dir"
build_target windows amd64 officecli-win-x64.exe        officecli-patch-win-x64.exe
build_target windows arm64 officecli-win-arm64.exe      officecli-patch-win-arm64.exe
build_target linux   amd64 officecli-linux-x64          officecli-patch-linux-x64
build_target linux   arm64 officecli-linux-arm64        officecli-patch-linux-arm64
build_target linux   amd64 officecli-linux-alpine-x64   officecli-patch-linux-alpine-x64 alpine
build_target linux   arm64 officecli-linux-alpine-arm64 officecli-patch-linux-alpine-arm64 alpine
build_target darwin  amd64 officecli-mac-x64            officecli-patch-mac-x64
build_target darwin  arm64 officecli-mac-arm64          officecli-patch-mac-arm64
