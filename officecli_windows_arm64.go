//go:build windows && arm64

package main

import _ "embed"

//go:embed tools/officecli-win-arm64.exe
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-win-arm64.exe"
