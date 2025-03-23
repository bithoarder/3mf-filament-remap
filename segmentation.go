package main

import (
	"fmt"
	"io"
	"strings"
)

const hextable = "0123456789abcdef"
const reverseHexTable = "" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\xff\xff\xff\xff\xff\xff" +
	"\xff\x0a\x0b\x0c\x0d\x0e\x0f\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\x0a\x0b\x0c\x0d\x0e\x0f\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
	"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff"

type Triangle struct {
	Color       int
	SpecialSide int
	Sub         [4]*Triangle
}

func (t *Triangle) String() string {
	if t.Sub[0] == nil && t.Sub[1] == nil && t.Sub[2] == nil && t.Sub[3] == nil {
		return fmt.Sprintf("%d", t.Color)
	} else {
		if t.Color != 0 {
			panic("invalid color")
		}
		sb := strings.Builder{}
		sb.WriteRune('(')
		for i, sub := range t.Sub {
			if i > 0 {
				sb.WriteByte(',')
			}
			if sub == nil {
				sb.WriteRune('-')
			} else {
				sb.WriteString(sub.String())
			}
		}
		sb.WriteRune(')')
		return sb.String()
	}
}

func (t *Triangle) SVG() string {
	sb := strings.Builder{}

	width := 1000.0
	height := 1000.0

	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(&sb, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%f\" height=\"%f\">\n", width, height)

	colors := []string{
		"#f00", "#0f0", "#00f", "#ff0", "#f0f", "#0ff",
		"#800", "#080", "#008", "#880", "#808", "#088",
		"#f88", "#8f8", "#88f", "#ff8", "#f8f", "#8ff",
	}

	var f func(t *Triangle, v [3][2]float64)
	f = func(t *Triangle, v [3][2]float64) {
		if t.Sub[0] == nil {
			if t.Color > 0 {
				fmt.Fprintf(&sb, "<polygon points=\"%f,%f %f,%f %f,%f\" fill=\"%s\" stroke=\"none\"/>\n",
					v[0][0], v[0][1], v[1][0], v[1][1], v[2][0], v[2][1], colors[t.Color-1])
			} else {
				// fmt.Fprintf(&sb, "<polygon points=\"%f,%f %f,%f %f,%f\" fill=\"none\" stroke=\"black\"/>\n",
				// 	v[0][0], v[0][1], v[1][0], v[1][1], v[2][0], v[2][1])
			}
		} else {
			// fmt.Fprintf(&sb, "<polygon points=\"%f,%f %f,%f %f,%f\" fill=\"none\" stroke=\"black\"/>\n",
			// 	v[0][0], v[0][1], v[1][0], v[1][1], v[2][0], v[2][1])

			t01 := [2]float64{(v[0][0] + v[1][0]) * 0.5, (v[0][1] + v[1][1]) * 0.5}
			t12 := [2]float64{(v[1][0] + v[2][0]) * 0.5, (v[1][1] + v[2][1]) * 0.5}
			t20 := [2]float64{(v[2][0] + v[0][0]) * 0.5, (v[2][1] + v[0][1]) * 0.5}

			// There might be a pattern to these rules...
			if t.Sub[2] == nil {
				if t.SpecialSide == 0 {
					f(t.Sub[0], [3][2]float64{t12, v[2], v[0]})
					f(t.Sub[1], [3][2]float64{v[0], v[1], t12})
				} else if t.SpecialSide == 1 {
					f(t.Sub[0], [3][2]float64{t20, v[0], v[1]})
					f(t.Sub[1], [3][2]float64{v[1], v[2], t20})
				} else if t.SpecialSide == 2 {
					f(t.Sub[0], [3][2]float64{t01, v[1], v[2]})
					f(t.Sub[1], [3][2]float64{v[2], v[0], t01})
				} else {
					panic("invalid special side")
				}
			} else if t.Sub[3] == nil {
				if t.SpecialSide == 0 {
					f(t.Sub[0], [3][2]float64{v[1], v[2], t20})
					f(t.Sub[1], [3][2]float64{t01, v[1], t20})
					f(t.Sub[1], [3][2]float64{v[0], t01, t20})
				} else if t.SpecialSide == 1 {
					f(t.Sub[0], [3][2]float64{v[2], v[0], t01})
					f(t.Sub[1], [3][2]float64{t12, v[2], t01})
					f(t.Sub[1], [3][2]float64{v[1], t12, t01})
				} else if t.SpecialSide == 2 {
					f(t.Sub[0], [3][2]float64{v[0], v[1], t12})
					f(t.Sub[1], [3][2]float64{t20, v[0], t12})
					f(t.Sub[1], [3][2]float64{v[2], t20, t12})
				} else {
					panic("invalid special side")
				}
			} else {
				// verified
				f(t.Sub[0], [3][2]float64{t01, t12, t20})
				f(t.Sub[1], [3][2]float64{t12, v[2], t20})
				f(t.Sub[2], [3][2]float64{t01, v[1], t12})
				f(t.Sub[3], [3][2]float64{v[0], t01, t20})
			}
		}
		fmt.Fprintf(&sb, "<polygon points=\"%f,%f %f,%f %f,%f\" fill=\"none\" stroke=\"#000\" stroke-opacity=\"0.1\"/>\n",
			v[0][0], v[0][1], v[1][0], v[1][1], v[2][0], v[2][1])
	}

	f(t, [3][2]float64{
		{width * 0.9, height * 0.9},
		{width * 0.9, height * 0.1},
		{width * 0.1, height * 0.1},
	})

	sb.WriteString("</svg>\n")
	return sb.String()
}

func (t *Triangle) AsSegmentation() string {
	encoding := []byte{}

	putNibble := func(c int) {
		encoding = append(encoding, hextable[c])
	}

	var f func(t *Triangle)
	f = func(t *Triangle) {
		if t.Sub[0] == nil {
			if t.Color < 3 {
				putNibble(int(t.Color << 2))
			} else {
				putNibble(0b1100)
				putNibble(int(t.Color - 3))
			}
		} else if t.Sub[2] == nil {
			putNibble(int(t.SpecialSide<<2 | 0b01))
			f(t.Sub[0])
			f(t.Sub[1])
		} else if t.Sub[3] == nil {
			putNibble(int(t.SpecialSide<<2 | 0b10))
			f(t.Sub[0])
			f(t.Sub[1])
			f(t.Sub[2])
		} else {
			putNibble(0b11)
			f(t.Sub[0])
			f(t.Sub[1])
			f(t.Sub[2])
			f(t.Sub[3])
		}
	}

	f(t)

	// reverse
	for i := 0; i < len(encoding)/2; i++ {
		encoding[i], encoding[len(encoding)-1-i] = encoding[len(encoding)-1-i], encoding[i]
	}

	return string(encoding)
}

func (t *Triangle) RemapColors(remap []int) {
	var f func(t *Triangle)
	f = func(t *Triangle) {
		if int(t.Color) < len(remap) {
			t.Color = remap[t.Color]
		}
		for i := 0; i < 4; i++ {
			if t.Sub[i] != nil {
				f(t.Sub[i])
			}
		}
	}
	f(t)
}

// ParseSegmentation decodes triangle painting as saved by Prusa and Orca slicer.
// ref: https://github.com/prusa3d/PrusaSlicer/blob/68c4bd671cddc20df1013c2181e640012de73b9c/src/libslic3r/TriangleSelector.cpp#L1872
// ref: https://github.com/prusa3d/PrusaSlicer/blob/68c4bd671cddc20df1013c2181e640012de73b9c/src/libslic3r/TriangleSelector.cpp#L1455
func ParseSegmentation(strEncoding string) (*Triangle, error) {
	i := len(strEncoding) - 1
	getNibble := func() (int, error) {
		if i < 0 {
			return 0, io.EOF
		}
		for {
			c := strEncoding[i]
			i -= 1
			if c != ' ' {
				return int(reverseHexTable[c]), nil
			}
		}
	}

	var f func() (*Triangle, error)
	f = func() (*Triangle, error) {
		if code, err := getNibble(); err != nil {
			return nil, err
		} else {
			t := Triangle{}
			numOfSplitSides := code & 0b11
			if numOfSplitSides == 0 {
				color := code >> 2
				if color == 3 {
					if nib, err := getNibble(); err != nil {
						return nil, err
					} else {
						color = nib + 3
					}
				}
				t.Color = color
			} else {
				numOfChildren := numOfSplitSides + 1
				specialSide := code >> 2
				t.SpecialSide = specialSide
				for i := 0; i < numOfChildren; i++ {
					if subTri, err := f(); err != nil {
						return nil, err
					} else {
						t.Sub[i] = subTri
					}
				}
			}

			return &t, nil
		}
	}

	return f()
}
