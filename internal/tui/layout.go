package tui

type panelRegion struct {
	pane scrollPane
	x0   int
	y0   int
	x1   int
	y1   int
}

func (r panelRegion) contains(x, y int) bool {
	return x >= r.x0 && x < r.x1 && y >= r.y0 && y < r.y1
}

type clickRegion struct {
	action string
	index  int
	x0     int
	y0     int
	x1     int
	y1     int
}

func (r clickRegion) contains(x, y int) bool {
	return x >= r.x0 && x < r.x1 && y >= r.y0 && y < r.y1
}
