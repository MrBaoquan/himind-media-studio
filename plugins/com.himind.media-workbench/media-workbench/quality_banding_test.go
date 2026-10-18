package main

import (
	"image"
	"image/color"
	"testing"
)

func TestAnalyzeBandingDetectsFlatBands(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: 40, B: 40, A: 255})
		}
	}
	report := analyzeBanding(img, 0.08, 12)
	if report.OK {
		t.Fatalf("flat image must fail banding check: %+v", report)
	}
	if len(report.Findings) == 0 {
		t.Fatal("flat image must produce findings")
	}
}

func TestAnalyzeBandingAcceptsSmoothGradient(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			value := uint8(10 + (x+y)/4)
			img.Set(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
		}
	}
	report := analyzeBanding(img, 0.08, 12)
	if !report.OK {
		t.Fatalf("smooth gradient should pass banding check: %+v", report)
	}
}
