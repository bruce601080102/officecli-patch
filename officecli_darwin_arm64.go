//go:build darwin && arm64

package main

import _ "embed"

//go:embed tools/officecli-mac-arm64
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-mac-arm64"
