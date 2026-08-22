//go:build screenshots

// Package screenshots writes deterministic website images from tcell's
// in-memory simulation screen.
package screenshots

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
)

const (
	cellWidth    = 9.2
	cellHeight   = 21.0
	frameInset   = 16.0
	chromeHeight = 38.0
)

// WriteSVG saves the current physical screen contents inside a terminal frame.
func WriteSVG(
	screen tcell.SimulationScreen,
	path string,
	title string,
) error {
	cells, width, height := screen.GetContents()
	if width < 1 || height < 1 || len(cells) < width*height {
		return fmt.Errorf("capture screen: invalid %dx%d contents", width, height)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create screenshot directory: %w", err)
	}

	canvasWidth := 2*frameInset + float64(width)*cellWidth
	canvasHeight := chromeHeight + float64(height)*cellHeight + frameInset
	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.1f %.1f" role="img" aria-label="%s">`, canvasWidth, canvasHeight, canvasWidth, canvasHeight, html.EscapeString(title))
	svg.WriteString(`<defs><filter id="shadow" x="-20%" y="-20%" width="140%" height="160%"><feDropShadow dx="0" dy="18" stdDeviation="20" flood-color="#000" flood-opacity=".35"/></filter><clipPath id="frame"><rect width="100%" height="100%" rx="13"/></clipPath></defs>`)
	svg.WriteString(`<g filter="url(#shadow)"><rect width="100%" height="100%" rx="13" fill="#111827" stroke="#34405a"/><g clip-path="url(#frame)">`)
	fmt.Fprintf(&svg, `<rect x="0" y="%.1f" width="100%%" height="%.1f" fill="#2e3440"/>`, chromeHeight, canvasHeight-chromeHeight)
	svg.WriteString(`<circle cx="17" cy="19" r="4" fill="#bf616a"/><circle cx="31" cy="19" r="4" fill="#d6ad72"/><circle cx="45" cy="19" r="4" fill="#8fbc8f"/>`)
	fmt.Fprintf(&svg, `<text x="62" y="23" fill="#71809b" font-family="ui-sans-serif,system-ui,sans-serif" font-size="10">%s</text>`, html.EscapeString(title))

	for row := 0; row < height; row++ {
		writeBackgrounds(&svg, cells[row*width:(row+1)*width], row)
		writeText(&svg, cells[row*width:(row+1)*width], row)
	}
	svg.WriteString(`</g></g></svg>`)

	if err := os.WriteFile(path, []byte(svg.String()), 0o644); err != nil {
		return fmt.Errorf("write screenshot: %w", err)
	}
	return nil
}

func writeBackgrounds(svg *strings.Builder, cells []tcell.SimCell, row int) {
	for start := 0; start < len(cells); {
		_, background, _ := cells[start].Style.Decompose()
		end := start + 1
		for end < len(cells) {
			_, next, _ := cells[end].Style.Decompose()
			if color(next, "#2e3440") != color(background, "#2e3440") {
				break
			}
			end++
		}
		fill := color(background, "#2e3440")
		if fill != "#2e3440" {
			fmt.Fprintf(svg, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`, frameInset+float64(start)*cellWidth, chromeHeight+float64(row)*cellHeight, float64(end-start)*cellWidth, cellHeight, fill)
		}
		start = end
	}
}

func writeText(svg *strings.Builder, cells []tcell.SimCell, row int) {
	for start := 0; start < len(cells); {
		foreground, _, attributes := cells[start].Style.Decompose()
		end := start + 1
		for end < len(cells) {
			nextForeground, _, nextAttributes := cells[end].Style.Decompose()
			if color(nextForeground, "#eceff4") != color(foreground, "#eceff4") || nextAttributes != attributes {
				break
			}
			end++
		}

		var content strings.Builder
		for _, cell := range cells[start:end] {
			if len(cell.Runes) == 0 {
				content.WriteByte(' ')
				continue
			}
			content.WriteRune(cell.Runes[0])
			for _, combining := range cell.Runes[1:] {
				content.WriteRune(combining)
			}
		}
		text := content.String()
		if strings.TrimSpace(text) != "" {
			weight := "400"
			if attributes&tcell.AttrBold != 0 {
				weight = "700"
			}
			decoration := ""
			if attributes&tcell.AttrUnderline != 0 {
				decoration = ` text-decoration="underline"`
			}
			fmt.Fprintf(svg, `<text xml:space="preserve" x="%.1f" y="%.1f" fill="%s" font-family="SFMono-Regular,Menlo,Consolas,Liberation Mono,monospace" font-size="15" font-weight="%s"%s>%s</text>`, frameInset+float64(start)*cellWidth, chromeHeight+float64(row)*cellHeight+16, color(foreground, "#eceff4"), weight, decoration, html.EscapeString(text))
		}
		start = end
	}
}

func color(value tcell.Color, fallback string) string {
	red, green, blue := value.RGB()
	if red < 0 || green < 0 || blue < 0 {
		return fallback
	}
	return fmt.Sprintf("#%02x%02x%02x", red, green, blue)
}
