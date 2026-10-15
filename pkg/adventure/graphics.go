package adventure

import (
	"bytes"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

type Color struct {
	R, G, B, A float64
}

func RGB(r, g, b float64) Color {
	return Color{r, g, b, 1}
}

type transform struct {
	tx, ty, sx, sy float64
}

type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

var (
	target     *ebiten.Image
	current    = Color{1, 1, 1, 1}
	lineWidth  = 1.0
	xf         = transform{0, 0, 1, 1}
	xfStack    []transform
	font       *text.GoTextFace
	bigFont    *text.GoTextFace
	activeFont *text.GoTextFace
)

func loadFonts() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	font = &text.GoTextFace{Source: source, Size: 14}
	bigFont = &text.GoTextFace{Source: source, Size: 30}
	activeFont = font
}

func (c Color) rgba() color.NRGBA {
	clamp := func(v float64) uint8 {
		return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
	}
	return color.NRGBA{clamp(c.R), clamp(c.G), clamp(c.B), clamp(c.A)}
}

func SetColor(r, g, b float64, a ...float64) {
	alpha := 1.0
	if len(a) > 0 {
		alpha = a[0]
	}
	current = Color{r, g, b, alpha}
}

func setColor(c Color) {
	current = c
}

func SetLineWidth(w float64) {
	lineWidth = w
}

func Push() {
	xfStack = append(xfStack, xf)
}

func Pop() {
	xf = xfStack[len(xfStack)-1]
	xfStack = xfStack[:len(xfStack)-1]
}

func Translate(x, y float64) {
	xf.tx += x * xf.sx
	xf.ty += y * xf.sy
}

func Scale(sx, sy float64) {
	xf.sx *= sx
	xf.sy *= sy
}

func point(x, y float64) (float32, float32) {
	return float32(xf.tx + x*xf.sx), float32(xf.ty + y*xf.sy)
}

func box(x, y, w, h float64) (float32, float32, float32, float32) {
	x0, y0 := point(x, y)
	x1, y1 := point(x+w, y+h)
	return min(x0, x1), min(y0, y1), max(x0, x1) - min(x0, x1), max(y0, y1) - min(y0, y1)
}

func FillRect(x, y, w, h float64) {
	bx, by, bw, bh := box(x, y, w, h)
	vector.FillRect(target, bx, by, bw, bh, current.rgba(), false)
}

func LineRect(x, y, w, h float64) {
	bx, by, bw, bh := box(x, y, w, h)
	vector.StrokeRect(target, bx, by, bw, bh, float32(lineWidth), current.rgba(), false)
}

func FillRoundedRect(x, y, w, h, r float64) {
	FillRect(x+r, y, w-2*r, h)
	FillRect(x, y+r, w, h-2*r)
	FillCircle(x+r, y+r, r)
	FillCircle(x+w-r, y+r, r)
	FillCircle(x+r, y+h-r, r)
	FillCircle(x+w-r, y+h-r, r)
}

func FillCircle(x, y, radius float64) {
	cx, cy := point(x, y)
	vector.FillCircle(target, cx, cy, float32(radius*math.Abs(xf.sy)), current.rgba(), true)
}

func LineCircle(x, y, radius float64) {
	cx, cy := point(x, y)
	vector.StrokeCircle(target, cx, cy, float32(radius*math.Abs(xf.sy)), float32(lineWidth), current.rgba(), true)
}

func Line(x0, y0, x1, y1 float64) {
	ax, ay := point(x0, y0)
	bx, by := point(x1, y1)
	vector.StrokeLine(target, ax, ay, bx, by, float32(lineWidth), current.rgba(), true)
}

func FillPolygon(coords ...float64) {
	var path vector.Path
	for i := 0; i+1 < len(coords); i += 2 {
		px, py := point(coords[i], coords[i+1])
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(current.rgba())
	vector.FillPath(target, &path, nil, op)
}

func fontHeight(face *text.GoTextFace) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent + m.HLineGap
}

func Print(s string, x, y float64) {
	px, py := point(x, y)
	op := &text.DrawOptions{}
	op.GeoM.Translate(math.Round(float64(px)), math.Round(float64(py)))
	op.ColorScale.ScaleWithColor(current.rgba())
	text.Draw(target, s, activeFont, op)
}

func wrapText(face *text.GoTextFace, s string, width float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			candidate := line + " " + word
			if text.Advance(candidate, face) > width {
				lines = append(lines, line)
				line = word
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func Printf(s string, x, y, width float64, align Align) {
	height := fontHeight(activeFont)
	for i, line := range wrapText(activeFont, s, width) {
		lx := x
		switch align {
		case AlignCenter:
			lx = x + (width-text.Advance(line, activeFont))/2
		case AlignRight:
			lx = x + width - text.Advance(line, activeFont)
		}
		Print(line, lx, y+float64(i)*height)
	}
}
