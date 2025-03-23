package main

import (
	"fmt"
	"log/slog"
	"os"
)

type SvgCmd struct {
	Segmentation string `arg:"" name:"seg" help:"Segmentation rule as used by Prusa/Bambu slicer to color triangles. Example: \"1C0C843\""`
	Dst          string `arg:"" name:"dst" help:"Destination svg file" type:"path"`
}

func (cmd *SvgCmd) Run() error {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	if triangle, err := ParseSegmentation(cmd.Segmentation); err != nil {
		return fmt.Errorf("failed to parse segmentation rule: %v", err)
	} else if err := os.WriteFile(cmd.Dst, []byte(triangle.SVG()), 0644); err != nil {
		return fmt.Errorf("failed to write svg file: %v", err)
	} else {
		return nil
	}
}
