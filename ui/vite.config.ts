import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// 关键约束（见 docs/repositioning-architecture.md §11）：
// 1. base 必须是相对路径 —— 资源以 plugin-ui://localhost/<plugin>/<view>/<entry> 提供，
//    绝对路径 /assets/x.js 会被宿主的路径解析器当成 plugin_id。
// 2. outDir 不能是 dist/ —— 打包器排除 dist 与 node_modules，会被静默丢掉。
export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  build: {
    // 产物必须落在【插件包根】下的 ui/app：
    // 打包器只收集插件目录，dist/ 与 node_modules 会被排除。
    outDir: '../plugins/com.himind.media-workbench/media-workbench/ui/app',
    emptyOutDir: true,
    sourcemap: false,
    target: 'chrome110',
  },
});
