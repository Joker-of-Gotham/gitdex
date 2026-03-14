package tui

func (m Model) contentHeight() int {
	height := m.height - 2
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) columnWidths() (leftWidth, rightWidth int, narrow bool) {
	if m.width < 110 {
		return m.width, 0, true
	}
	gap := 1
	usable := m.width - gap
	rightWidth = usable * 2 / 5
	if rightWidth < 26 {
		rightWidth = 26
	}
	if rightWidth > m.width-32 {
		rightWidth = m.width - 32
	}
	leftWidth = usable - rightWidth
	if leftWidth < 40 {
		return m.width, 0, true
	}
	return leftWidth, rightWidth, false
}

func (m Model) rightPanelHeights(totalHeight int) (areasHeight, observabilityHeight int) {
	areasHeight = totalHeight * 42 / 100
	if areasHeight < 12 {
		areasHeight = 12
	}
	if areasHeight > totalHeight-12 {
		areasHeight = totalHeight - 12
	}
	observabilityHeight = totalHeight - 1 - areasHeight
	if observabilityHeight < 10 {
		observabilityHeight = 10
		areasHeight = totalHeight - 1 - observabilityHeight
	}
	return areasHeight, observabilityHeight
}

type layoutMetrics struct {
	workspaceHeight int
	logHeight       int
	logGap          int
}

func (m Model) computeLayoutMetrics() layoutMetrics {
	height := m.contentHeight()
	_, _, narrow := m.columnWidths()

	if m.width < 80 {
		remaining := height
		if remaining < 9 {
			remaining = 9
		}
		logHeight := 1
		logGap := 0
		if m.logExpanded {
			logHeight = maxInt(6, remaining/4)
			logGap = 1
		}
		mainHeight := remaining - logGap - logHeight
		if mainHeight < 8 {
			mainHeight = 8
		}
		obsHeight := maxInt(7, mainHeight/3)
		workspaceHeight := mainHeight - obsHeight
		if workspaceHeight < 4 {
			deficit := 4 - workspaceHeight
			reduceObs := minInt(deficit, maxInt(0, obsHeight-6))
			obsHeight -= reduceObs
			workspaceHeight = mainHeight - obsHeight
			if workspaceHeight < 4 {
				workspaceHeight = 4
			}
		}
		return layoutMetrics{workspaceHeight: workspaceHeight, logHeight: logHeight, logGap: logGap}
	}

	logHeight := 1
	logGap := 0
	if m.logExpanded {
		if narrow {
			logHeight = maxInt(8, height/3)
		} else {
			logHeight = maxInt(10, height/3)
		}
		logGap = 1
	}
	topHeight := height - logGap - logHeight
	if !narrow && topHeight < 12 {
		topHeight = 12
		logHeight = maxInt(1, height-logGap-topHeight)
	}
	workspaceHeight := topHeight
	if narrow && workspaceHeight < 4 {
		workspaceHeight = 4
	}
	return layoutMetrics{workspaceHeight: workspaceHeight, logHeight: logHeight, logGap: logGap}
}
