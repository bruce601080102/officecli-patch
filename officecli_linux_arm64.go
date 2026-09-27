//go:build linux && arm64 && !alpine

package main

import _ "embed"

//go:embed tools/officecli-linux-arm64
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-linux-arm64"
