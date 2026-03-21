package layout

import (
	"testing"
)

func TestClassify_Compact(t *testing.T) {
	d := Classify(80, 24)
	if d.Breakpoint != Compact {
		t.Errorf("expected Compact, got %v", d.Breakpoint)
	}
	if d.HeightClass != ShortHeight {
		t.Errorf("expected ShortHeight, got %v", d.HeightClass)
	}
}

func TestClassify_Standard(t *testing.T) {
	d := Classify(120, 40)
	if d.Breakpoint != Standard {
		t.Errorf("expected Standard, got %v", d.Breakpoint)
	}
	if d.HeightClass != NormalHeight {
		t.Errorf("expected NormalHeight, got %v", d.HeightClass)
	}
}

func TestClassify_Wide(t *testing.T) {
	d := Classify(160, 60)
	if d.Breakpoint != Wide {
		t.Errorf("expected Wide, got %v", d.Breakpoint)
	}
	if d.HeightClass != TallHeight {
		t.Errorf("expected TallHeight, got %v", d.HeightClass)
	}
}

func TestClassify_Boundaries(t *testing.T) {
	tests := []struct {
		w    int
		want Breakpoint
	}{
		{79, Compact},
		{80, Compact},
		{99, Compact},
		{100, Standard},
		{139, Standard},
		{140, Wide},
		{200, Wide},
	}
	for _, tt := range tests {
		d := Classify(tt.w, 40)
		if d.Breakpoint != tt.want {
			t.Errorf("Classify(%d, 40).Breakpoint = %v, want %v", tt.w, d.Breakpoint, tt.want)
		}
	}
}

func TestDimensions_MainWidth(t *testing.T) {
	wide := Classify(140, 40)
	if wide.MainWidth() == 0 {
		t.Error("wide MainWidth should not be 0")
	}

	compact := Classify(80, 40)
	if compact.MainWidth() != 80 {
		t.Errorf("compact MainWidth should equal total width, got %d", compact.MainWidth())
	}
}

func TestDimensions_ShowNav(t *testing.T) {
	if Classify(80, 40).ShowNav() {
		t.Error("compact should not show nav")
	}
	if Classify(120, 40).ShowNav() {
		t.Error("standard should not show nav")
	}
	if !Classify(160, 40).ShowNav() {
		t.Error("wide should show nav for the control-plane layout")
	}
}

func TestDimensions_ShowInspector(t *testing.T) {
	if Classify(80, 40).ShowInspector() {
		t.Error("compact should not show inspector")
	}
	if !Classify(120, 40).ShowInspector() {
		t.Error("standard should show inspector")
	}
	if !Classify(160, 40).ShowInspector() {
		t.Error("wide should show inspector")
	}
}

func TestDimensions_ContentHeight(t *testing.T) {
	d := Classify(120, 40)
	h := d.ContentHeight()
	if h <= 0 {
		t.Errorf("ContentHeight should be positive, got %d", h)
	}
	if h >= 40 {
		t.Errorf("ContentHeight should leave room for input, got %d for height 40", h)
	}
}

func TestDimensions_ContentHeight_Minimum(t *testing.T) {
	d := Classify(80, 5)
	if d.ContentHeight() < 5 {
		t.Errorf("ContentHeight should have minimum of 5, got %d", d.ContentHeight())
	}
}
