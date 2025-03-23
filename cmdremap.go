package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

type MaterialMap struct {
	From, To int
}

func (m *MaterialMap) UnmarshalText(text []byte) error {
	if parts := strings.Split(string(text), ":"); len(parts) == 2 {
		if from, err := strconv.Atoi(parts[0]); err == nil {
			if to, err := strconv.Atoi(parts[1]); err == nil {
				if from < 1 || from > 16 || to < 1 || to > 16 {
					return fmt.Errorf("invalid material remap: \"%s\", material indices bust be in the range 1-16", text)
				}
				*m = MaterialMap{From: from, To: to}
				return nil
			}
		}
	}
	return fmt.Errorf("invalid color map: \"%s\", expected format: \"<int>,<int>\"", text)
}

type RemapCmd struct {
	Src string        `arg:"" name:"src" help:"Source 3mf file" type:"existingfile"`
	Dst string        `arg:"" name:"dst" help:"Destination 3mf file" type:"path"`
	Map []MaterialMap `name:"map" help:"Material mapping. Format: <from-material-index>:<to-material-index>. Example: 1:2 to map existing material 1 to material 2."`
}

func (cmd *RemapCmd) Run() error {
	materialMap := make([]int, 17)
	for _, m := range cmd.Map {
		if materialMap[m.From] != 0 {
			return fmt.Errorf("material index %d is mapped to multiple material: %d and %d", m.From, materialMap[m.From], m.To)
		}
		materialMap[m.From] = m.To
	}
	for i := range materialMap {
		if materialMap[i] == 0 {
			materialMap[i] = i
		}
	}

	VPrintf("material map: %v\n", materialMap)

	srcZipArchive, err := zip.OpenReader(cmd.Src)
	if err != nil {
		return fmt.Errorf("failed to open 3mf file: %w", err)
	}
	defer srcZipArchive.Close()

	dstZipFile, err := os.Create(cmd.Dst)
	if err != nil {
		return fmt.Errorf("failed to create 3mf file: %w", err)
	}
	defer dstZipFile.Close()

	dstZipArchive := zip.NewWriter(dstZipFile)
	defer dstZipArchive.Close()

	// This makes a bunch of assumptions about the layout of 3mf files.

	for _, f := range srcZipArchive.File {
		VPrintf("processing %s\n", f.Name)
		if r, err := f.Open(); err != nil {
			return fmt.Errorf("failed to open file %s %s: %w", cmd.Src, f.Name, err)
		} else if fileContent, err := io.ReadAll(r); err != nil {
			return fmt.Errorf("failed to read file %s %s: %w", cmd.Src, f.Name, err)
		} else {
			if strings.HasPrefix(f.Name, "3D/") && strings.HasSuffix(f.Name, ".model") {
				fileContent, err = remapModelColors(fileContent, materialMap)
			} else if f.Name == "Metadata/Slic3r_PE_model.config" {
				fileContent, err = remapMetadata(fileContent, materialMap, true)
			} else if f.Name == "Metadata/model_settings.config" {
				fileContent, err = remapMetadata(fileContent, materialMap, false)
			} else if f.Name == "Metadata/project_settings.config" {
				fileContent, err = remapOrcaProjectSettings(fileContent, materialMap)
			} else if f.Name == "Metadata/Slic3r_PE.config" {
				fileContent, err = remapPrusaProjectSettings(fileContent, materialMap)
			}
			if err != nil {
				return fmt.Errorf("failed to remap file %s %s: %w", cmd.Src, f.Name, err)
			}

			if w, err := dstZipArchive.Create(f.Name); err != nil {
				return fmt.Errorf("failed to add file %s %s: %w", cmd.Dst, f.Name, err)
			} else if _, err := w.Write(fileContent); err != nil {
				return fmt.Errorf("failed to write file %s %s: %w", cmd.Dst, f.Name, err)
			}
			_ = r.Close()
		}
	}

	return nil
}

func remapModelColors(content []byte, remap []int) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(content); err != nil {
		return nil, err
	}

	remappedTriangles := 0

	knownSegmentationAttrs := []string{
		"mmu_segmentation", // Prusa Slicer
		"paint_color",      // Orca Slicer
	}

	for _, triangle := range doc.FindElements("./model/resources/object/mesh/triangles/triangle") {
		for _, attrName := range knownSegmentationAttrs {
			if paintColorAttr := triangle.SelectAttr(attrName); paintColorAttr != nil {
				segmentation := paintColorAttr.Value
				if tri, err := ParseSegmentation(segmentation); err != nil {
					return nil, fmt.Errorf("failed to parse color segmentation: %w", err)
				} else {
					tri.RemapColors(remap)
					newSegmentation := tri.AsSegmentation()
					if segmentation != newSegmentation {
						remappedTriangles += 1
						paintColorAttr.Value = newSegmentation
					}
				}
			}
		}
	}

	VPrintf("  remapped %d triangles\n", remappedTriangles)

	return doc.WriteToBytes()
}

// remapMetadata remaps the extruder index used by objects.
func remapMetadata(content []byte, remap []int, prusaMetadata bool) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(content); err != nil {
		return nil, err
	}

	remappedObjects := 0

	for _, metadata := range doc.FindElements("./config/object/metadata") {
		if metadata.SelectAttrValue("key", "") == "extruder" && (!prusaMetadata || metadata.SelectAttrValue("type", "") == "object") {
			if valueAttr := metadata.SelectAttr("value"); valueAttr != nil {
				if value, err := strconv.Atoi(valueAttr.Value); err != nil {
					return nil, fmt.Errorf("failed to parse metadata extruder value %s: %w", metadata.FullTag(), err)
				} else if value >= 0 && value < len(remap) && value != remap[value] {
					remappedObjects += 1
					valueAttr.Value = strconv.Itoa(remap[value])
				}
			}
		}
	}

	for _, metadata := range doc.FindElements("./config/object/volume/metadata") {
		if metadata.SelectAttrValue("key", "") == "extruder" && (!prusaMetadata || metadata.SelectAttrValue("type", "") == "object") {
			if valueAttr := metadata.SelectAttr("value"); valueAttr != nil {
				if value, err := strconv.Atoi(valueAttr.Value); err != nil {
					return nil, fmt.Errorf("failed to parse metadata extruder value %s: %w", metadata.FullTag(), err)
				} else if value >= 0 && value < len(remap) && value != remap[value] {
					remappedObjects += 1
					valueAttr.Value = strconv.Itoa(remap[value])
				}
			}
		}
	}

	VPrintf("  remapped %d objects\n", remappedObjects)

	return doc.WriteToBytes()

	// slic3rpe:mmu_segmentation
	// paint_color
}

// remapOrcaProjectSettings remaps the filament colour.
// Other filament related settings to avoid breaking with future versions of Orca that might have additional material related arrays.
func remapOrcaProjectSettings(content []byte, remap []int) ([]byte, error) {
	settings := map[string]any{}
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, err
	}

	colors := settings["filament_colour"].([]any)
	newColors := slices.Clone(colors)
	for i, r := range remap[1:] {
		r -= 1
		if i < len(colors) && r >= 0 && r < len(colors) && r != i {
			newColors[r] = colors[i]
		}
	}
	settings["filament_colour"] = newColors

	return json.MarshalIndent(settings, "", "  ")
}

// ; extruder_colour = #FF8000;#DB5182;#3EC0FF;#FF4F4F;#FBEB7D
var prusaProjectExtruderColorsRx = regexp.MustCompile(`(?s)^(.*?\n;\s*extruder_colour\s*=\s*)((?:#[0-9a-fA-F]{6};?)+)(\n.*)$`)

func remapPrusaProjectSettings(content []byte, remap []int) ([]byte, error) {
	groups := prusaProjectExtruderColorsRx.FindStringSubmatch(string(content))
	if len(groups) != 4 {
		VPrintf("  skipping, failed to parse extruder colors\n")
		return content, nil
	}

	colors := strings.Split(groups[2], ";")
	newColors := slices.Clone(colors)
	for i, r := range remap[1:] {
		r -= 1
		if i < len(colors) && r >= 0 && r < len(colors) && r != i {
			newColors[r] = colors[i]
		}
	}

	return []byte(groups[1] + strings.Join(newColors, ";") + groups[3]), nil
}
