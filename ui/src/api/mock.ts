import type {ConstraintPayload, ProjectDetail, ProjectList} from '../types';

// 浏览器里跑 UI 时的假数据。字段形状与真实能力返回一致，
// 这样切到宿主时不需要改组件。
const now = new Date().toISOString();

const lines = [
  {id: 'hening', name: '合宁客专', color: '#FF6B6B', openYear: 2008, lengthKm: 166, coords: [[117.28, 31.86], [117.6, 31.95], [118.0, 32.02], [118.4, 32.08], [118.78, 32.06]]},
  {id: 'jinghu', name: '京沪高铁', color: '#E74C3C', openYear: 2011, lengthKm: 323, coords: [[117.17, 33.72], [117.38, 32.88], [117.84, 32.58], [118.31, 32.2], [118.8, 31.98]]},
  {id: 'hefu', name: '合福高铁', color: '#2ECC71', openYear: 2015, lengthKm: 360, coords: [[117.25, 31.86], [117.3, 31.4], [117.6, 31.0], [117.9, 30.6], [118.1, 30.2], [118.3, 29.9]]},
  {id: 'shanghehang', name: '商合杭高铁', color: '#F39C12', openYear: 2020, lengthKm: 576, coords: [[115.8, 33.9], [116.4, 33.6], [117.0, 33.2], [117.5, 32.6], [117.9, 32.0], [118.4, 31.4], [118.9, 30.9]]},
  {id: 'chihuang', name: '池黄高铁', color: '#AB47BC', openYear: 2024, lengthKm: 125, coords: [[117.5, 30.66], [117.8, 30.5], [118.0, 30.35], [118.15, 30.2]]},
];

const mockShotTemplates = [
  {id: 'big-number', name: '数字冲击', family: 'stat', status: 'ready', duration_seconds: 5, storyboard_shot: true, storyboard_order: 10},
  {id: 'compare', name: '数据对比', family: 'compare', status: 'ready', duration_seconds: 7, storyboard_shot: true, storyboard_order: 20},
  {id: 'quote', name: '引语卡', family: 'quote', status: 'ready', duration_seconds: 6, storyboard_shot: true, storyboard_order: 30},
  {id: 'bar-race', name: '时序竞速', family: 'race', status: 'ready', duration_seconds: 12, storyboard_shot: true, storyboard_order: 40},
  {id: 'timeline', name: '时间线', family: 'timeline', status: 'ready', duration_seconds: 12, storyboard_shot: true, storyboard_order: 50},
  {id: 'choropleth', name: '分级填色地图', family: 'choropleth', status: 'ready', duration_seconds: 10, storyboard_shot: true, storyboard_order: 60},
  {id: 'composition', name: '占比构成', family: 'composition', status: 'ready', duration_seconds: 10, storyboard_shot: true, storyboard_order: 70},
  {id: 'flow', name: '流程链路', family: 'flow', status: 'ready', duration_seconds: 10, storyboard_shot: true, storyboard_order: 80},
  {id: 'route-growth', name: '路网生长', family: 'map-transit', status: 'ready', duration_seconds: 18, storyboard_shot: true, storyboard_order: 90},
];

const projects: ProjectDetail[] = [
  {
    project: {
      id: 'anhui-hsr', name: '安徽高铁网络', template_id: 'route-growth', style_id: 'tech-blue',
      revision: 3, updated_at: now, status: 'draft',
      recipe: {
        template_id: 'route-growth', template_version: '1.0.0', style_id: 'tech-blue', style_version: '1.1.0',
        adapter: 'remotion', title: '安徽省高铁网络', subtitle: '17 条线路 · 市市通高铁', period: '2008 - 2025',
        duration_seconds: 18, canvas: {width: 2160, height: 3840, fps: 30},
        safe_area: {top: 460, right: 108, bottom: 268, left: 108},
        data: {region: {name: '安徽省'}, totalMileage: 2972, lines},
      },
    },
    jobs: [{job_id: 'render-1', status: 'completed', progress: 100, message: 'Remotion 视频已生成', stage: 'render'}],
    artifacts: [{artifact_id: 'video-artifact-1', kind: 'video/mp4', path: 'artifacts/v1.mp4', size: 3148758, sha256: '884944fc', ready: true, created_at: now}],
  },
  {
    project: {
      id: 'storyboard-demo', name: '多镜头演示', template_id: 'storyboard', style_id: 'tech-blue',
      revision: 1, updated_at: now, status: 'draft',
      recipe: {
        template_id: 'storyboard', style_id: 'tech-blue', adapter: 'remotion',
        title: '多镜头演示', duration_seconds: 8, canvas: {width: 2160, height: 3840, fps: 30},
        data: {storyboard: [
          {shot: 'big-number', duration_seconds: 4, data: {value: 2972, unit: '公里', label: '安徽高铁总里程'}},
          {shot: 'quote', duration_seconds: 4, data: {text: '交通运输是国民经济的基础性、先导性、战略性产业。', author: '《交通强国建设纲要》'}},
        ]},
      },
    },
    jobs: [], artifacts: [],
  },
  {
    project: {
      id: 'hefei-gdp', name: '合肥 GDP 排行', template_id: 'leaderboard-tech', style_id: 'culture-ink',
      revision: 1, updated_at: now, status: 'draft',
      recipe: {
        template_id: 'leaderboard-tech', style_id: 'culture-ink', adapter: 'remotion',
        title: '合肥各区 GDP', duration_seconds: 12, canvas: {width: 2160, height: 3840, fps: 30},
        data: {items: []},
      },
    },
    jobs: [], artifacts: [],
  },
];

const rule = (code: string, level: string, stage: string, message: string, kind: string, id: string) => ({
  code, level, stage, message, enforced_by: {kind, id}, resolved: true, detail: '已解析',
});

const rules = [
      rule('SAFE_AREA', 'error', 'recipe', '安全区必须有效且不重叠', 'capability', 'media.workbench.recipe.validate'),
      rule('DURATION_RANGE', 'error', 'recipe', '时长 1–300 秒', 'capability', 'media.workbench.recipe.validate'),
      rule('STATION_ANCHORED', 'error', 'geometry', '站点到轨迹 ≤5 km', 'capability', 'media.workbench.route.validate'),
      rule('NO_VISIBLE_SPIKE', 'error', 'geometry', '不能有原路返回尖刺', 'capability', 'media.workbench.route.validate'),
      rule('NO_LOOP', 'error', 'geometry', '轨迹不能回到先前位置', 'capability', 'media.workbench.route.validate'),
      rule('ADJACENT_GAP', 'warn', 'geometry', '相邻点间距 ≤30 km', 'capability', 'media.workbench.route.validate'),
      rule('RENDER_APPROVAL', 'gate', 'render', '渲染前需人工确认', 'runtime', 'approval_center'),
      rule('DETERMINISTIC_OUTPUT', 'error', 'render', '两次渲染逐字节相同', 'script', 'dev/verify-render.py'),
      rule('CAPABILITY_NAMESPACE', 'error', 'extension', '能力 ID 落在声明命名空间下', 'test', 'TestManifestDeclaresNamespaceAndDistributionTargets'),
  rule('FALLBACK_MATCHES_CATALOG', 'error', 'extension', '兜底与目录一致', 'test', 'TestFallbackTemplatesMatchCatalog'),
];

const constraints: ConstraintPayload = {
  constraints: {id: 'media-workbench.standard', version: '1.0.0', rules},
  resolved: rules,
  summary: {rules: rules.length, resolved: rules.length, unresolved: 0, by_level: {error: 8, warn: 1, gate: 1}, by_kind: {capability: 5, runtime: 1, script: 1, test: 2}},
  unresolved: [],
};

function toBase64Utf8(text: string): string {
  const bytes = new TextEncoder().encode(text);
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary);
}

const MOCK_PREVIEW_HTML = '<!doctype html><html><body style="margin:0;background:#0a0f1f;color:#e7eaf0;font-family:Microsoft YaHei,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh"><div style="text-align:center"><div style="font-size:20px;font-weight:700">动效预览（mock）</div><div style="margin-top:8px;font-size:12px;opacity:.6">浏览器模式下不运行 Remotion Player</div></div></body></html>';

export async function mockCapability<T>(id: string, input: Record<string, unknown>): Promise<T> {
  await new Promise((r) => setTimeout(r, 100));
  switch (id) {
    case 'media.workbench.project.list':
      return {
        items: projects.map((p) => ({
          id: p.project.id, name: p.project.name, template_id: p.project.template_id,
          style_id: p.project.style_id, revision: p.project.revision,
          updated_at: p.project.updated_at, status: p.project.status,
        })),
        total: projects.length, workspace_root: 'D:/develop/video-flow（示例）',
      } as T;
    case 'media.workbench.project.get': {
      const found = projects.find((p) => p.project.id === input.project_id);
      if (!found) throw new Error('project not found');
      return found as T;
    }
    case 'media.workbench.constraint.list':
      return constraints as T;
    case 'media.workbench.preview.player':
      return {artifact: {artifact_id: 'mock-preview-player', kind: 'preview/player'}} as T;
    case 'media.workbench.artifact.read':
      return {artifact: {artifact_id: input.artifact_id}, size: MOCK_PREVIEW_HTML.length, base64: toBase64Utf8(MOCK_PREVIEW_HTML)} as T;
    case 'media.workbench.catalog.list':
      return {
        templates: [
          ...mockShotTemplates,
          {id: 'leaderboard-tech', family: 'leaderboard', name: '排行榜 · 科技蓝', status: 'ready'},
          {id: 'storyboard', family: 'storyboard', name: '多镜头成片', status: 'ready', duration_seconds: 14},
          {id: 'gdp-map-story', family: 'map-story', name: '地图数据故事', status: 'placeholder'},
        ],
        styles: [{id: 'tech-blue', name: '科技蓝'}, {id: 'culture-ink', name: '徽墨'}],
        adapters: [{id: 'remotion'}, {id: 'hyperframes'}],
      } as T;
    case 'media.workbench.template.schema': {
      const templateId = String(input.template_id ?? '');
      const template = mockShotTemplates.find((item) => item.id === templateId);
      if (!template) throw new Error(`mock 未实现镜头: ${templateId}`);
      const defaults: Record<string, Record<string, unknown>> = {
        'big-number': {value: 2972, unit: '公里', label: '安徽高铁总里程', delta: '+12.4%'},
        compare: {unit: '亿元', left: {label: '合肥', value: 12674}, right: {label: '芜湖', value: 5121}, verdict: '合肥约为芜湖的 2.5 倍'},
        quote: {text: '交通运输是国民经济的基础性、先导性、战略性产业。', author: '《交通强国建设纲要》', role: '2019'},
        composition: {unit: '亿元', total_label: '文旅消费', categories: [{label: '交通', value: 2860}, {label: '住宿', value: 2450}]},
        flow: {conclusion: '一条流水线，从想法到成片', steps: [{label: '洞察'}, {label: '脚本'}]},
        timeline: {unit: '年', events: [{year: 2008, title: '合宁客专开通'}, {year: 2025, title: '市市通高铁'}]},
        choropleth: {geo: {$ref: 'plugin:catalog/data/geo/anhui-cities.geojson'}, values: {}},
        'bar-race': {unit: '亿元', years: {start: 2015, end: 2025}, items: []},
        'route-growth': {region: {name: '安徽省'}, lines: [], totalMileage: 0},
      };
      return {template, schema: null, default_data: defaults[templateId] ?? {}} as T;
    }
    default:
      throw new Error(`mock 未实现能力: ${id}`);
  }
}


