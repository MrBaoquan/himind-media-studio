package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

// style_pack.go — Style Pack 的静态契约校验。
//
// project.import 只负责登记；stylepack.validate 负责在登记前检查
// style-pack.json、render.mjs 固定 CLI、分镜与输出路径。

func stylePackValidate(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	sourceRoot, err := workspaceFile(root, in.SourceDir)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("source_dir: " + err.Error())
	}
	info, err := os.Stat(sourceRoot)
	if err != nil || !info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("source_dir 不存在或不是目录")
	}
	pack, err := readCodeStylePack(sourceRoot)
	if err != nil {
		return map[string]any{"valid": false, "issues": []map[string]any{issue("style_pack_manifest", "style", err.Error(), "修复 style-pack.json")}, "retryable": false}, nil
	}
	checks := make([]map[string]any, 0, 10)
	issues := []map[string]any{}
	check := func(id string, ok bool, detail, code, message, remediation string) {
		checks = append(checks, map[string]any{"id": id, "passed": ok, "detail": detail})
		if !ok {
			issues = append(issues, issue(code, "style", message, remediation))
		}
	}
	check("schema", pack.Schema == "himind.video.style-pack.v1", pack.Schema, "style_pack_schema", "style-pack schema 必须为 himind.video.style-pack.v1", "修复 style-pack.json")
	check("renderer", pack.Renderer == "browser-code", pack.Renderer, "style_pack_renderer", "renderer 必须为 browser-code", "设置 renderer=browser-code")
	check("duration", pack.DurationSeconds > 0 && pack.DurationSeconds <= 300, "", "style_pack_duration", "duration_seconds 必须在 1–300 之间", "修复 duration_seconds")
	networkMode := pack.Network.Mode
	if networkMode == "" {
		networkMode = "none"
	}
	networkOK := networkMode == "none" || networkMode == "declared" || networkMode == "full"
	check("network_mode", networkOK, networkMode, "style_pack_network", "network.mode 必须为 none / declared / full", "修复 network.mode")
	if networkMode == "declared" {
		check("network_allow", len(pack.Network.Allow) > 0, strings.Join(pack.Network.Allow, ","), "style_pack_network_allow", "declared 模式必须声明 allow 域名", "补充 network.allow")
	}
	if networkMode == "full" {
		check("network_trust", pack.Trust == "signed", pack.Trust, "style_pack_network_full", "full 网络模式只允许 signed Style Pack", "默认使用 none，或提供签名信任")
	}
	if pack.Delivery.Web {
		check("delivery_web_width", pack.Delivery.WebWidth >= 160 && pack.Delivery.WebWidth <= 2160, "", "style_pack_web_width", "delivery.web_width 必须在 160–2160 之间", "修复 delivery.web_width")
		check("delivery_web_crf", pack.Delivery.WebCRF >= 18 && pack.Delivery.WebCRF <= 32, "", "style_pack_web_crf", "delivery.web_crf 必须在 18–32 之间", "修复 delivery.web_crf")
	}
	if pack.Delivery.MaxMasterMB < 0 {
		check("delivery_max_master", false, "", "style_pack_max_master", "delivery.max_master_mb 不能为负数", "修复 delivery.max_master_mb")
	}
	if pack.Runtime.MaxWorkers < 0 || pack.Runtime.MaxWorkers > 8 {
		check("runtime_max_workers", false, fmt.Sprintf("%d", pack.Runtime.MaxWorkers), "style_pack_max_workers", "runtime.max_workers 必须在 1–8 之间（0 表示不限制）", "修复 runtime.max_workers")
	}

	entryPath, entryErr := workspaceFile(sourceRoot, pack.Entry)
	entryOK := entryErr == nil && fileExists(entryPath)
	check("entry", entryOK, pack.Entry, "style_pack_entry", "入口文件不存在", "检查 entry")
	if entryOK {
		raw, _ := os.ReadFile(entryPath)
		text := string(raw)
		for _, flag := range []string{"--sheet", "--strip", "--stills", "--frames", "--encode"} {
			check("cli:"+flag, strings.Contains(text, flag), flag, "style_pack_cli", "render.mjs 未声明固定 CLI: "+flag, "实现或保留固定子命令")
		}
	}

	check("package_json", fileExists(filepath.Join(sourceRoot, "package.json")), "package.json", "style_pack_package", "缺少 package.json", "添加依赖声明")
	check("storyboard", fileExists(filepath.Join(sourceRoot, "STORYBOARD.md")), "STORYBOARD.md", "style_pack_storyboard", "缺少 STORYBOARD.md", "先写分镜再登记项目")

	for name, output := range map[string]string{
		"video_out":  pack.VideoOut,
		"sheet_out":  pack.Preview.SheetOut,
		"strip_out":  pack.Preview.StripOut,
		"stills_out": pack.Preview.StillsOut,
	} {
		ok := strings.TrimSpace(output) != "" && !filepath.IsAbs(filepath.FromSlash(output)) && !strings.HasPrefix(filepath.Clean(filepath.FromSlash(output)), "..")
		check("output:"+name, ok, output, "style_pack_output", "输出路径必须是 source_dir 内相对路径", "修复 "+name)
	}

	return map[string]any{
		"valid":      len(issues) == 0,
		"style_pack": pack,
		"source_dir": in.SourceDir,
		"checks":     checks,
		"issues":     issues,
		"note":       "valid=true 后再调用 project.import；本项目只执行固定 render.mjs CLI。",
	}, nil
}
