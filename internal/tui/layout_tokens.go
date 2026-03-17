package tui

// layoutTokens defines spacing/grid/breakpoint behavior for the main shell layout.
type layoutTokens struct {
	headerH      int
	inputH       int
	minContentH  int
	leftRatioPct int
	minLeftW     int
	minRightW    int
	minPanelH    int
	gitRatioPct  int
	goalRatioPct int
}

func layoutTokensForWidth(width int) layoutTokens {
	// default (desktop)
	t := layoutTokens{
		headerH:      2,
		inputH:       3,
		minContentH:  5,
		leftRatioPct: 65,
		minLeftW:     20,
		minRightW:    15,
		minPanelH:    1,
		gitRatioPct:  30,
		goalRatioPct: 35,
	}
	if width > 0 && width < 100 {
		// compact viewport: give right column slightly more width for details.
		t.leftRatioPct = 60
	}
	if width > 0 && width < 80 {
		// narrow viewport: avoid right panel starvation.
		t.minRightW = 20
		t.minLeftW = 18
	}
	return t
}

