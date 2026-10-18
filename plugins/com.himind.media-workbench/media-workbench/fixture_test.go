package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// fixture_test.go —— 回归基线的"完整性"门禁。
//
// 这一步不渲染（渲染在 dev/regress-stills.ps1 里做），只检查：
//   1. 清单引用的 recipe 与基线 PNG 都存在；
//   2. 基线尺寸与清单声明一致；
//   3. 每个 ready 模板都有基线（否则新镜头会悄悄逃过回归）；
//   4. 阈值在合理区间。
// 这样 `go test ./...` 仍然很快，但基线腐烂会立刻暴露。

type fixtureManifest struct {
	Schema      string  `json:"schema"`
	Version     string  `json:"version"`
	SSIMFloor   float64 `json:"ssimFloor"`
	DiffYAvgMax float64 `json:"diffYAvgMax"`
	DiffYMaxMax float64 `json:"diffYMaxMax"`
	Size        struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"size"`
	Fixtures []struct {
		ID          string `json:"id"`
		Template    string `json:"template"`
		Composition string `json:"composition"`
		Recipe      string `json:"recipe"`
		Frames      []int  `json:"frames"`
	} `json:"fixtures"`
}

func loadFixtures(t *testing.T) fixtureManifest {
	t.Helper()
	path := filepath.Join(pluginRoot(), "catalog", "fixtures", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse fixture manifest: %v", err)
	}
	return manifest
}

func pngDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()
	header := make([]byte, 24)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, 0, err
	}
	if string(header[1:4]) != "PNG" {
		return 0, 0, fmt.Errorf("not a PNG file")
	}
	return int(binary.BigEndian.Uint32(header[16:20])), int(binary.BigEndian.Uint32(header[20:24])), nil
}

func TestGoldenFixturesAreComplete(t *testing.T) {
	manifest := loadFixtures(t)
	root := filepath.Join(pluginRoot(), "catalog", "fixtures")

	if manifest.Schema != "himind.video.fixtures.v1" {
		t.Errorf("unexpected fixture schema: %s", manifest.Schema)
	}
	if manifest.Size.Width <= 0 || manifest.Size.Height <= 0 {
		t.Fatalf("fixture size is invalid: %dx%d", manifest.Size.Width, manifest.Size.Height)
	}
	if manifest.SSIMFloor <= 0.5 || manifest.SSIMFloor > 1 {
		t.Errorf("ssimFloor out of range: %v", manifest.SSIMFloor)
	}
	if manifest.DiffYAvgMax <= 0 || manifest.DiffYMaxMax <= 0 {
		t.Errorf("diff thresholds must be positive (avg=%v max=%v)", manifest.DiffYAvgMax, manifest.DiffYMaxMax)
	}

	templates := map[string]videoTemplate{}
	for _, item := range loadTemplates() {
		templates[item.ID] = item
	}

	indexSource, _ := os.ReadFile(filepath.Join(pluginRoot(), "renderer", "remotion", "index.jsx"))
	covered := map[string]bool{}

	for _, fixture := range manifest.Fixtures {
		covered[fixture.Template] = true

		recipePath := filepath.Join(root, filepath.FromSlash(fixture.Recipe))
		if _, err := os.Stat(recipePath); err != nil {
			t.Errorf("%s: missing recipe snapshot %s", fixture.ID, fixture.Recipe)
		}

		if tpl, ok := templates[fixture.Template]; ok {
			if tpl.Status != "ready" {
				t.Errorf("%s: fixture references template %s with status %q (expected ready)", fixture.ID, fixture.Template, tpl.Status)
			}
			if tpl.Composition != fixture.Composition {
				t.Errorf("%s: composition mismatch: fixture=%s catalog=%s", fixture.ID, fixture.Composition, tpl.Composition)
			}
		} else {
			t.Errorf("%s: template %s is not in the catalog", fixture.ID, fixture.Template)
		}

		if len(indexSource) > 0 && !containsString(string(indexSource), `id="`+fixture.Composition+`"`) {
			t.Errorf("%s: composition %q is not registered in index.jsx", fixture.ID, fixture.Composition)
		}

		if len(fixture.Frames) == 0 {
			t.Errorf("%s: no frames declared", fixture.ID)
		}
		for _, frame := range fixture.Frames {
			baseline := filepath.Join(root, fixture.ID, fmt.Sprintf("frame-%04d.png", frame))
			w, h, err := pngDimensions(baseline)
			if err != nil {
				t.Errorf("%s: baseline missing for frame %d (%v); run dev/regress-stills.ps1 -Update", fixture.ID, frame, err)
				continue
			}
			if w != manifest.Size.Width || h != manifest.Size.Height {
				t.Errorf("%s frame %d: baseline is %dx%d, manifest says %dx%d",
					fixture.ID, frame, w, h, manifest.Size.Width, manifest.Size.Height)
			}
		}
	}

	// 每个 ready 模板都必须有基线，否则新镜头会悄悄逃过回归。
	for _, tpl := range templates {
		if tpl.Status == "ready" && !covered[tpl.ID] {
			t.Errorf("ready template %s has no golden fixture; add it to catalog/fixtures/manifest.json", tpl.ID)
		}
	}

	t.Logf("%d fixtures cover %d ready templates", len(manifest.Fixtures), len(covered))
}

func containsString(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
