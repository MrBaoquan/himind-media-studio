export interface ProjectSummary {
  id: string;
  name: string;
  template_id?: string;
  style_id?: string;
  mode?: 'template' | 'code' | string;
  renderer?: string;
  style_pack?: string;
  style_version?: string;
  source_dir?: string;
  entry?: string;
  source_hash?: string;
  preview_status?: string;
  determinism_status?: string;
  determinism_hash?: string;
  duration_seconds?: number;
  revision: number;
  updated_at: string;
  workspace_root?: string;
  status?: string;
}

export interface ProjectList {
  items: ProjectSummary[];
  total: number;
  workspace_root: string;
}

/**
 * Recipe 是「一条视频」的完整输入文档。
 *
 * 注意 data 的类型：**故意不写任何题材字段**。
 * 创作台不认识 lines / items / media —— 那是模板的事。
 * 外壳只处理通用部分：版本引用、画布、安全区、时长、文案。
 */
export interface Recipe {
  template_id: string;
  template_version?: string;
  style_id: string;
  style_version?: string;
  adapter?: string;
  title?: string;
  subtitle?: string;
  period?: string;
  unit?: string;
  source?: string;
  duration_seconds: number;
  canvas: {width: number; height: number; fps: number};
  safe_area?: {top: number; right: number; bottom: number; left: number};
  style_tokens?: Record<string, unknown>;
  data?: Record<string, unknown>;
}

export interface FeedbackEntry {
  id: string;
  category: string;
  feedback: string;
  accepted?: boolean;
  status?: string;
  status_note?: string;
  resolved_at?: string;
  created_at: string;
}

export interface Artifact {
  artifact_id: string;
  kind: string;
  path: string;
  size: number;
  sha256: string;
  ready: boolean;
  quality_status?: string;
  quality_report?: {ok?: boolean; warnings?: string[]};
  created_at: string;
}

export interface Job {
  job_id: string;
  status: string;
  progress: number;
  message?: string;
  stage?: string;
}

export interface ProjectDetail {
  project: ProjectSummary & {recipe?: Recipe};
  mode?: 'template' | 'code' | string;
  storyboard?: string;
  feedback?: FeedbackEntry[];
  plan?: VideoPlan | null;
  jobs?: Job[];
  artifacts?: Artifact[];
}

export interface ConstraintRule {
  code: string;
  level: 'error' | 'warn' | 'gate' | 'info' | string;
  stage: string;
  message: string;
  enforced_by: {kind: string; id: string};
  resolved?: boolean;
  detail?: string;
}

export interface ConstraintPayload {
  constraints: {id: string; version: string; rules: ConstraintRule[]};
  resolved: ConstraintRule[];
  summary: {
    rules: number;
    resolved: number;
    unresolved: number;
    by_level: Record<string, number>;
    by_kind: Record<string, number>;
  };
  unresolved: string[];
}
/** 模板目录项。Storyboard 编辑器只消费这里声明的 storyboard_* 字段。 */
export interface CatalogOption {
  id: string;
  name?: string;
  status?: string;
  family?: string;
  composition?: string;
  duration_seconds?: number;
  data_contract?: string;
  storyboard_shot?: boolean;
  storyboard_order?: number;
}

/** template.schema 的返回：契约 + 新建镜头时的最小示例数据。 */
export interface TemplateSchemaPayload {
  template: CatalogOption;
  schema?: Record<string, unknown> | null;
  default_data?: Record<string, unknown>;
}
/** Plan 是 AI 与确定性执行之间的规划契约。 */
export interface PlanKeyPoint {
  claim: string;
  evidence_ref?: string;
}

export interface PlanShot {
  shot: string;
  role?: string;
  duration_seconds: number;
  data_binding?: string;
  rationale?: string;
  title?: string;
  subtitle?: string;
}

export interface VideoPlan {
  version: string;
  brief?: {
    goal?: string;
    audience?: string;
    platform?: string;
    duration_seconds?: number;
    tone?: string;
    must_include?: string[];
    must_avoid?: string[];
    data_refs?: string[];
  };
  narrative?: {
    thesis?: string;
    structure?: string;
    key_points?: PlanKeyPoint[];
    required_data?: string[];
    constraints?: string[];
    risk_notes?: string[];
  };
  storyboard?: {shots?: PlanShot[]};
  presentation?: {
    title?: string;
    subtitle?: string;
    source?: string;
    style_id?: string;
    transition_frames?: number;
  };
}

export interface AssetResource {
  id: string;
  type: string;
  path: string;
  format?: string;
  mime?: string;
  role?: string;
  runtime?: string;
  entry?: string;
  description?: string;
}

export interface AssetPack {
  schema: string;
  id: string;
  version: string;
  name: string;
  description?: string;
  license?: string;
  resources: AssetResource[];
  entrypoints?: Array<{id: string; resource: string; runtime: string; entry: string}>;
  consumers?: string[];
  tags?: string[];
  path?: string;
  source?: string;
  installed?: boolean;
  install_path?: string;
}

export interface AssetMarketEntry {
  id: string;
  name?: string;
  version: string;
  file_name: string;
  file_size: number;
  sha256: string;
  download_url: string;
  license?: string;
  status?: string;
  tags?: string[];
  installed?: boolean;
  installed_version?: string;
  update_available?: boolean;
}

export interface AssetMarket {
  schema: string;
  name?: string;
  version?: string;
  repository?: string;
  assets: AssetMarketEntry[];
  total: number;
}

