package main

import (
	"fmt"
	"runtime"
)

var banner = `
---------------------------------------------------------
                        supergoctl
Version:	%v
BuildTime:	%v
GitHash:	%v
GOVersion:	%v
---------------------------------------------------------
`

//VERSION
var (
	VERSION    = "0.0.0"
	BUILD_DATE = "0000-00-00 00:00:00"
	GitHash    = "000000"
)

func PrintVersion() {
	fmt.Printf(banner, VERSION, BUILD_DATE, GitHash, runtime.Version())
}
