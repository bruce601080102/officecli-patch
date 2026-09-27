//go:build linux && amd64 && !alpine

package main

import _ "embed"

//go:embed tools/officecli-linux-x64
var embeddedOfficeCLIBinary []byte

const embeddedOfficeCLIFilename = "officecli-linux-x64"
