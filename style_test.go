package main

import "testing"

func TestScaleUI(t *testing.T) {
	css := `.card {
  font-size: 13px;
  padding: 22px 24px 20px;
  border: 1px solid;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.32);
  letter-spacing: 0.01em;
}`
	want := `.card {
  font-size: 20.8px;
  padding: 35.2px 38.4px 32px;
  border: 1.6px solid;
  box-shadow: 0 12.8px 44.8px rgba(0, 0, 0, 0.32);
  letter-spacing: 0.01em;
}`
	if got := scaleUI(css, 160); got != want {
		t.Fatalf("scaleUI():\n%s\nwant:\n%s", got, want)
	}
}

func TestScaleUIAndText(t *testing.T) {
	css := `.card { font-size: 13px; min-height: 36px; }`
	want := `.card { font-size: 26px; min-height: 57.6px; }`
	if got := scaleText(scaleUI(css, 160), 125); got != want {
		t.Fatalf("scaleText(scaleUI()) = %q, want %q", got, want)
	}
}

func TestScaleUIAt100(t *testing.T) {
	css := `.card { font-size: 13px; padding: 22px; }`
	if got := scaleUI(css, 100); got != css {
		t.Fatalf("scaleUI() = %q, want unchanged CSS", got)
	}
}

func TestScaleForDPI(t *testing.T) {
	tests := []struct {
		dpi  float64
		want int
	}{
		{0, 100},
		{80, 100},
		{96, 100},
		{101.6, 100},
		{162.56, 160},
		{192, 200},
		{300, 200},
	}
	for _, tt := range tests {
		if got := scaleForDPI(tt.dpi); got != tt.want {
			t.Errorf("scaleForDPI(%.2f) = %d, want %d", tt.dpi, got, tt.want)
		}
	}
}
