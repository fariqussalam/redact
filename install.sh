#!/bin/sh
# Install redact – the local streaming text redactor.
# Usage: curl -fsSL https://raw.githubusercontent.com/fariqussalam/redact/master/install.sh | sh

set -eu

REPO="fariqussalam/redact"
BINARY="redact"

detect_os() {
	case "$(uname -s)" in
		Linux*)  echo "linux" ;;
		Darwin*) echo "darwin" ;;
		*)       echo "unknown" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
		x86_64|amd64)       echo "amd64" ;;
		aarch64|arm64)      echo "arm64" ;;
		*)                  echo "unknown" ;;
	esac
}

fatal() {
	echo "Error: $*" >&2
	exit 1
}

has_curl() { command -v curl >/dev/null 2>&1; }
has_wget() { command -v wget >/dev/null 2>&1; }

fetch() {
	if has_curl; then
		curl -fsSL "$@"
	elif has_wget; then
		wget -qO- "$@"
	else
		fatal "need curl or wget to download files"
	fi
}

download() {
	url="$1"
	out="$2"
	if has_curl; then
		curl -fsSL "$url" -o "$out"
	elif has_wget; then
		wget -q "$url" -O "$out"
	else
		fatal "need curl or wget to download files"
	fi
}

latest_tag() {
	tag=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
		| grep '"tag_name"' \
		| cut -d '"' -f 4 \
		| head -n1)
	[ -n "$tag" ] || fatal "could not fetch latest release tag from GitHub API"
	echo "$tag"
}

strip_v() {
	printf '%s' "$1" | sed 's/^v//'
}

main() {
	os=$(detect_os)
	arch=$(detect_arch)

	[ "$os" = "unknown" ] && fatal "unsupported OS: $(uname -s)"
	[ "$arch" = "unknown" ] && fatal "unsupported architecture: $(uname -m)"

	tag=$(latest_tag)
	echo "  Detected: $os/$arch"
	echo "  Release:  $tag"
	version=$(strip_v "$tag")
	archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
	archive_url="https://github.com/$REPO/releases/download/$tag/$archive"
	checksums_url="https://github.com/$REPO/releases/download/$tag/checksums.txt"

	# Work in a temp directory
	tmpdir=$(mktemp -d) || fatal "could not create temp directory"
	cd "$tmpdir"

	# Download archive
	echo "  Downloading $archive ..."
	download "$archive_url" archive.tar.gz
	set +e
	expected=$(fetch "$checksums_url" 2>/dev/null \
		| grep -F "$archive" \
		| awk '{print $1}')
	set -e
	if [ -n "$expected" ]; then
		actual=$(sha256sum archive.tar.gz 2>/dev/null | awk '{print $1}' || shasum -a 256 archive.tar.gz | awk '{print $1}')
		if [ "$expected" != "$actual" ]; then
			fatal "checksum mismatch for $archive"
		fi
		echo "  Checksum verified."
	fi

	# Extract
	tar xzf archive.tar.gz || fatal "could not extract archive"

	if [ -n "${REDACT_INSTALL_DIR:-}" ]; then
		dest="$REDACT_INSTALL_DIR"
		mkdir -p "$dest"
	elif [ -w /usr/local/bin ]; then
		dest="/usr/local/bin"
	elif [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
		dest="$HOME/.local/bin"
	elif [ -d "$HOME/go/bin" ] && [ -w "$HOME/go/bin" ]; then
		dest="$HOME/go/bin"
	elif mkdir -p "$HOME/.local/bin" 2>/dev/null && [ -w "$HOME/.local/bin" ]; then
		dest="$HOME/.local/bin"
	else
		dest="/usr/local/bin"
		echo "  Installing to $dest (requires sudo) ..."
		sudo mkdir -p "$dest"
	fi

	if [ -w "$dest" ]; then
		mv "$BINARY" "$dest/"
	else
		echo "  Installing to $dest (requires sudo) ..."
		sudo mv "$BINARY" "$dest/"
	fi

	cd / && rm -rf "$tmpdir"

	echo "  Installed $BINARY to $dest"
	echo ""
	echo "  Run '$BINARY --help' to get started."
	echo "  Usage:  some_command 2>&1 | $BINARY"
}

main
