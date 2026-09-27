//go:build windows && amd64

package main

import _ "embed"

//go:embed tools/officecli-win-x64.exe
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-win-x64.exe"
