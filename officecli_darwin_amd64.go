//go:build darwin && amd64

package main

import _ "embed"

//go:embed tools/officecli-mac-x64
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-mac-x64"
