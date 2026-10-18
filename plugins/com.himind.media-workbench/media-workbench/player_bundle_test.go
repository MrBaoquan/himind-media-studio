package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPlayerBundleFingerprintTracksRendererSources 守住播放器缓存的失效条件。
//
// 回归背景：新增 Shot 后 index.jsx 已更新，但旧 player.bundle.js 仍命中缓存，
// 预览里显示"未知的 composition"。指纹必须覆盖所有会进入 bundle 的源码。
func TestPlayerBundleFingerprintTracksRendererSources(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(relative, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("player.jsx", "import {COMPOSITIONS} from './index.jsx';\n")
	mustWrite("index.jsx", "export const COMPOSITIONS = {A: 1};\n")
	mustWrite("package.json", `{"name":"runtime"}`)
	mustWrite("shots/A.jsx", "export const A = () => null;\n")

	first, err := playerBundleFingerprint(root)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	if first == "" {
		t.Fatal("fingerprint is empty")
	}

	mustWrite("shots/A.jsx", "export const A = () => 'changed';\n")
	second, err := playerBundleFingerprint(root)
	if err != nil {
		t.Fatalf("fingerprint after source change: %v", err)
	}
	if second == first {
		t.Fatal("fingerprint did not change after a Shot source changed; stale player bundle would be reused")
	}

	// node_modules、锁文件和旧产物都不应影响指纹。
	mustWrite("node_modules/pkg/index.js", "module.exports = 1;\n")
	mustWrite("package-lock.json", `{"lockfileVersion":3}`)
	mustWrite("player.bundle.js", "old bundle")
	third, err := playerBundleFingerprint(root)
	if err != nil {
		t.Fatalf("fingerprint after ignored files: %v", err)
	}
	if third != second {
		t.Fatal("fingerprint changed for files that do not affect the player bundle")
	}
}
