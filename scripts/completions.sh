#!/bin/sh
# Generates shell completion scripts for packaging into release archives.
set -e
rm -rf completions
mkdir completions
for shell in fish bash zsh; do
	go run ./cmd/harvest completion "$shell" >"completions/harvest.$shell"
done
