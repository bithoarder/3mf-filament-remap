package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var verbose = false

func Printf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func VPrintf(format string, a ...interface{}) {
	if verbose {
		Printf(format, a...)
	}
}

var cli struct {
	Verbose bool     `help:"Enable debug output."`
	Remap   RemapCmd `cmd:"" help:"Remaps color indices in a 3mf file."`
	Svg     SvgCmd   `cmd:"" help:"Creates an SVG file from a segmentation rule."`
}

func main() {
	ctx := kong.Parse(&cli)
	verbose = cli.Verbose
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
