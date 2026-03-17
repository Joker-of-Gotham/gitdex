package tui

const (
	scrollStepLine  = 1
	scrollStepWheel = 3
)

func (m Model) zoneFromXY(x, y int) FocusZone {
	geo := m.calcLayout()
	h := m.height
	if h <= 0 {
		h = 24
	}

	inputStartY := h - geo.inputH
	if inputStartY < 0 {
		inputStartY = 0
	}

	if y >= inputStartY {
		return FocusInput
	}
	if y < geo.headerH {
		return FocusInput
	}
	if x < geo.leftW {
		return FocusLeft
	}

	contentY := y - geo.headerH
	gitEnd := geo.gitH
	goalEnd := gitEnd + 1 + geo.goalH
	if contentY < gitEnd {
		return FocusGit
	}
	if contentY < goalEnd+1 {
		return FocusGoals
	}
	return FocusLog
}

func (m Model) cycleFocus() Model {
	order := []FocusZone{FocusInput, FocusLeft, FocusGit, FocusGoals, FocusLog}
	for i, z := range order {
		if z == m.focusZone {
			next := order[(i+1)%len(order)]
			m.focusZone = next
			m.composerFocus = (next == FocusInput)
			return m
		}
	}
	m.focusZone = FocusInput
	m.composerFocus = true
	return m
}

func (m *Model) applyScrollDelta(zone FocusZone, delta int) {
	if zone == FocusInput {
		return
	}
	m.panelScrolls[zone] += delta
	if m.panelScrolls[zone] < 0 {
		m.panelScrolls[zone] = 0
	}
}

