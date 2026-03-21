package layout

type Breakpoint int

const (
	Compact  Breakpoint = iota // 80-99 columns: single column
	Standard                   // 100-139 columns: two columns
	Wide                       // 140+ columns: three columns
)

type HeightClass int

const (
	ShortHeight  HeightClass = iota // <32 rows
	NormalHeight                    // 32-51 rows
	TallHeight                      // 52+ rows
)

type Dimensions struct {
	Width       int
	Height      int
	Breakpoint  Breakpoint
	HeightClass HeightClass
}

func Classify(width, height int) Dimensions {
	return Dimensions{
		Width:       width,
		Height:      height,
		Breakpoint:  classifyWidth(width),
		HeightClass: classifyHeight(height),
	}
}

func classifyWidth(w int) Breakpoint {
	switch {
	case w >= 140:
		return Wide
	case w >= 100:
		return Standard
	default:
		return Compact
	}
}

func classifyHeight(h int) HeightClass {
	switch {
	case h >= 52:
		return TallHeight
	case h >= 32:
		return NormalHeight
	default:
		return ShortHeight
	}
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (d Dimensions) MainWidth() int {
	width := d.Width - d.NavWidth() - d.InspectorWidth()
	if d.ShowNav() {
		width--
	}
	if d.ShowInspector() {
		width--
	}
	if width < 48 {
		return 48
	}
	return width
}

func (d Dimensions) NavWidth() int {
	if !d.ShowNav() {
		return 0
	}
	return clamp(d.Width*22/100, 26, 34)
}

func (d Dimensions) InspectorWidth() int {
	switch d.Breakpoint {
	case Wide:
		return clamp(d.Width*26/100, 34, 46)
	case Standard:
		return clamp(d.Width*32/100, 30, 42)
	default:
		return 0
	}
}

func (d Dimensions) HeaderHeight() int    { return 2 }
func (d Dimensions) StatusBarHeight() int { return 1 }
func (d Dimensions) ComposerHeight() int  { return 3 }

func (d Dimensions) ContentHeight() int {
	usable := d.Height - d.HeaderHeight() - d.StatusBarHeight() - d.ComposerHeight()
	if usable < 5 {
		return 5
	}
	return usable
}

func (d Dimensions) ShowNav() bool {
	return d.Breakpoint == Wide
}

func (d Dimensions) ShowInspector() bool {
	return d.Breakpoint >= Standard
}
