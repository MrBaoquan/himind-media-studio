// techniques/sankey.js —— 桑基图布局。
//
// 纯函数：输入 links + 画布高度，输出节点、连线位置和厚度。
// 端点坐标、标签和动画都由 Shot 层决定；这里只做确定性布局。

/**
 * @param {{source:string,target:string,value:number}[]} links
 * @param {{height:number,gap?:number,minValue?:number}} options
 */
export function buildSankeyLayout(links, options = {}) {
  const height = Math.max(1, Number(options.height) || 1);
  const gap = Math.max(0, Number(options.gap) || 0);
  const minValue = Math.max(0, Number(options.minValue) || 0);
  const clean = (Array.isArray(links) ? links : [])
    .map((link) => ({
      source: String(link && link.source || ''),
      target: String(link && link.target || ''),
      value: Number(link && link.value),
    }))
    .filter((link) => link.source && link.target && Number.isFinite(link.value) && link.value > minValue);

  if (!clean.length) {
    return {total: 0, nodes: {sources: [], targets: []}, links: [], scale: 0};
  }

  const total = clean.reduce((sum, link) => sum + link.value, 0);
  const sourceIds = uniqueInOrder(clean.map((link) => link.source));
  const targetIds = uniqueInOrder(clean.map((link) => link.target));
  const maxNodes = Math.max(sourceIds.length, targetIds.length);
  const flowHeight = Math.max(1, height - gap * Math.max(0, maxNodes - 1));
  const scale = flowHeight / total;

  const sourceValues = sumsById(clean, 'source');
  const targetValues = sumsById(clean, 'target');
  const sources = layoutColumn(sourceIds, sourceValues, {height, gap, scale});
  const targets = layoutColumn(targetIds, targetValues, {height, gap, scale});
  const sourceMap = new Map(sources.map((node) => [node.id, node]));
  const targetMap = new Map(targets.map((node) => [node.id, node]));
  const sourceOffset = new Map(sourceIds.map((id) => [id, 0]));
  const targetOffset = new Map(targetIds.map((id) => [id, 0]));

  const outbound = clean.map((link, index) => {
    const source = sourceMap.get(link.source);
    const target = targetMap.get(link.target);
    const thickness = link.value * scale;
    const sourceY = (source ? source.y : 0) + (sourceOffset.get(link.source) || 0);
    const targetY = (target ? target.y : 0) + (targetOffset.get(link.target) || 0);
    sourceOffset.set(link.source, (sourceOffset.get(link.source) || 0) + thickness);
    targetOffset.set(link.target, (targetOffset.get(link.target) || 0) + thickness);
    return {
      index,
      source: link.source,
      target: link.target,
      value: link.value,
      thickness,
      sourceY,
      targetY,
    };
  });

  return {total, nodes: {sources, targets}, links: outbound, scale};
}

function uniqueInOrder(values) {
  const seen = new Set();
  const out = [];
  for (const value of values) {
    if (!seen.has(value)) {
      seen.add(value);
      out.push(value);
    }
  }
  return out;
}

function sumsById(links, key) {
  const map = new Map();
  for (const link of links) {
    map.set(link[key], (map.get(link[key]) || 0) + link.value);
  }
  return map;
}

function layoutColumn(ids, values, {height, gap, scale}) {
  const nodes = ids.map((id) => {
    const value = values.get(id) || 0;
    return {id, value, height: value * scale};
  });
  const columnHeight = nodes.reduce((sum, node) => sum + node.height, 0) + gap * Math.max(0, nodes.length - 1);
  let cursor = Math.max(0, (height - columnHeight) / 2);
  for (const node of nodes) {
    node.y = cursor;
    cursor += node.height + gap;
  }
  return nodes;
}
