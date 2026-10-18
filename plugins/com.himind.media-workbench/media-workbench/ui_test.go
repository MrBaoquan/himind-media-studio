package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ui_test.go —— 守住"UI 产物真的能随包加载"这件事。
//
// 两个最容易踩、且本地看不出来的坑：
//   1. 构建产物落在 dist/ —— 打包器排除 dist，装完打开是白屏；
//   2. Vite 用绝对路径 /assets/x.js —— 宿主把 plugin-ui://localhost/<plugin>/<view>/<entry>
//      的第一段当成 plugin_id，解析失败。
// 这两条一旦破，本地开发完全正常，只有装到用户机器上才暴露。

func TestUIViewEntryIsBuiltAndRelative(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(pluginRoot(), "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	var manifest struct {
		Contributes struct {
			Views []struct {
				ID    string `json:"id"`
				Entry string `json:"entry"`
			} `json:"views"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}
	if len(manifest.Contributes.Views) == 0 {
		t.Fatal("plugin declares no views")
	}

	view := manifest.Contributes.Views[0]
	if !strings.HasSuffix(view.Entry, ".html") {
		t.Fatalf("view entry must be an .html file, got %q", view.Entry)
	}
	if strings.HasPrefix(view.Entry, "dist/") || strings.Contains(view.Entry, "/dist/") {
		t.Fatalf("view entry must not live under dist/ (excluded from packaging): %q", view.Entry)
	}

	entryPath := filepath.Join(pluginRoot(), filepath.FromSlash(view.Entry))
	content, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("view entry is not built (%s); run `npm run build` in ui/: %v", view.Entry, err)
	}

	html := string(content)
	// 必须引用相对资源；绝对路径会被宿主解析成 plugin_id。
	if strings.Contains(html, `src="/`) || strings.Contains(html, `href="/`) {
		t.Error("built index.html uses absolute asset paths; set Vite base to './'")
	}
	if !strings.Contains(html, "./assets/") {
		t.Error("built index.html does not reference ./assets/; the build output looks wrong")
	}

	// 资源文件必须真实存在（构建产物完整性）
	for _, marker := range []string{"./assets/"} {
		idx := strings.Index(html, marker)
		if idx < 0 {
			continue
		}
		rest := html[idx+len(marker):]
		end := strings.IndexAny(rest, `"'`)
		if end <= 0 {
			continue
		}
		asset := filepath.Join(filepath.Dir(entryPath), "assets", rest[:end])
		if _, statErr := os.Stat(asset); statErr != nil {
			t.Errorf("referenced asset is missing: %s", rest[:end])
		}
	}
}
