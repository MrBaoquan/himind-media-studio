import * as THREE from '../node_modules/three/build/three.module.js';

const W = 1080, H = 1920, FPS = 30, DURATION = 30;
const webglCanvas = document.getElementById('webgl');
const overlay = document.getElementById('overlay');
const ctx = overlay.getContext('2d');

const DEFAULT = {
  title: '为什么地球有四季？',
  subtitle: '30 秒看懂：不是离太阳远近',
  scenes: [
    {id:'orbit', title:'地球在公转', caption:'轨道接近圆形；日地距离变化不是四季主因'},
    {id:'tilt', title:'地轴倾斜 23.44°', caption:'地轴方向在一年中几乎保持不变'},
    {id:'north', title:'北半球夏至', caption:'北半球更朝向太阳：太阳更高、白昼更长'},
    {id:'south', title:'南半球夏至', caption:'南北半球季节相反'},
    {id:'conclusion', title:'四季 = 地轴倾斜 + 公转', caption:'不是离太阳远近'},
  ],
  facts: [
    {value:'23.44°', label:'地轴倾角'},
    {value:'0.0167', label:'轨道偏心率'},
    {value:'1.47–1.52 亿 km', label:'日地距离'},
  ],
  source: 'NASA Earth Fact Sheet · NOAA Seasons',
};
let DATA = DEFAULT;
const SCENE_WINDOWS = [
  {id:'orbit', start:0, end:6},
  {id:'tilt', start:6, end:12},
  {id:'north', start:12, end:20},
  {id:'south', start:20, end:26},
  {id:'conclusion', start:26, end:30},
];
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const sourceScenes = Array.isArray(value.scenes) ? value.scenes : [];
      const scenes = DEFAULT.scenes.map((fallback) => {
        const item = sourceScenes.find((x) => x && x.id === fallback.id) || {};
        return {id:fallback.id, title:String(item.title || fallback.title), caption:String(item.caption || fallback.caption)};
      });
      const facts = Array.isArray(value.facts) ? value.facts.slice(0, 3).map((x) => ({value:String(x.value || ''), label:String(x.label || '')})) : DEFAULT.facts;
      DATA = {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), scenes, facts:facts.length === 3 ? facts : DEFAULT.facts, source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
function sceneText(id) { return DATA.scenes.find((s) => s.id === id) || DEFAULT.scenes.find((s) => s.id === id); }
function sceneAt(t) { return SCENE_WINDOWS.find((s) => t >= s.start && t < s.end) || SCENE_WINDOWS[SCENE_WINDOWS.length - 1]; }
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {ink:'#eef8ff', muted:'rgba(238,248,255,.58)', dim:'rgba(238,248,255,.28)', cyan:'#63dcff', gold:'#ffd166', warm:'#ffb35c', green:'#5fe3b3', line:'rgba(160,220,255,.16)'};
const TILT = THREE.MathUtils.degToRad(23.44);
const AXIS = new THREE.Vector3(-Math.sin(TILT), Math.cos(TILT), 0).normalize();

const renderer = new THREE.WebGLRenderer({canvas: webglCanvas, antialias:true, preserveDrawingBuffer:true, powerPreference:'high-performance'});
renderer.setPixelRatio(1); renderer.setSize(W, H, false); renderer.outputColorSpace = THREE.SRGBColorSpace; renderer.toneMapping = THREE.ACESFilmicToneMapping; renderer.toneMappingExposure = 1.2;
const textureLoader = new THREE.TextureLoader();
function loadTexture(url, colorSpace) {
  return new Promise((resolve, reject) => {
    textureLoader.load(url, (texture) => {
      texture.colorSpace = colorSpace;
      texture.anisotropy = 8;
      resolve(texture);
    }, undefined, reject);
  });
}
const [earthDayTex, earthNightTex, earthCloudTex, earthSpecTex, earthNormalTex] = await Promise.all([
  loadTexture('../assets/earth_atmos_2048.jpg', THREE.SRGBColorSpace),
  loadTexture('../assets/earth_lights_2048.png', THREE.SRGBColorSpace),
  loadTexture('../assets/earth_clouds_1024.png', THREE.SRGBColorSpace),
  loadTexture('../assets/earth_specular_2048.jpg', THREE.NoColorSpace),
  loadTexture('../assets/earth_normal_2048.jpg', THREE.NoColorSpace),
]);
earthCloudTex.wrapS = THREE.RepeatWrapping;
const scene = new THREE.Scene(); scene.background = new THREE.Color(0x04070f); scene.fog = new THREE.Fog(0x04070f, 22, 48);
const camera = new THREE.PerspectiveCamera(38, W / H, 0.1, 120);

const nebulaUniforms = {time: {value: 0}};
const nebulaMaterial = new THREE.ShaderMaterial({
  uniforms: nebulaUniforms,
  vertexShader: `
    varying vec3 vDir;
    void main() {
      vDir = normalize(position);
      gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
    }
  `,
  fragmentShader: `
    uniform float time;
    varying vec3 vDir;
    float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
    float noise(vec2 p) {
      vec2 i = floor(p); vec2 f = fract(p);
      f = f * f * (3.0 - 2.0 * f);
      float a = hash(i);
      float b = hash(i + vec2(1.0, 0.0));
      float c = hash(i + vec2(0.0, 1.0));
      float d = hash(i + vec2(1.0, 1.0));
      return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
    }
    float fbm(vec2 p) {
      float v = 0.0; float a = 0.5;
      for (int i = 0; i < 4; i++) { v += a * noise(p); p *= 2.0; a *= 0.5; }
      return v;
    }
    void main() {
      float h = vDir.y * 0.5 + 0.5;
      vec3 top = vec3(0.004, 0.010, 0.026);
      vec3 bottom = vec3(0.010, 0.018, 0.045);
      vec3 col = mix(bottom, top, h);
      float band = exp(-pow((vDir.y - 0.12 * sin(vDir.x * 2.5)) * 2.8, 2.0));
      float n = fbm(vDir.xz * 2.2 + vec2(vDir.y * 1.7, time * 0.002));
      col += vec3(0.012, 0.025, 0.065) * band * (0.35 + 0.65 * n);
      col += (hash(gl_FragCoord.xy) - 0.5) / 255.0;
      gl_FragColor = vec4(col, 1.0);
      #include <tonemapping_fragment>
      #include <colorspace_fragment>
    }
  `,
  side: THREE.BackSide,
  depthWrite: false,
  depthTest: false,
});
const nebula = new THREE.Mesh(new THREE.SphereGeometry(72, 64, 48), nebulaMaterial);
nebula.renderOrder = -10;
scene.add(nebula);
scene.add(new THREE.AmbientLight(0x23384d, 1.4));
const sunLight = new THREE.PointLight(0xffd27a, 4.8, 32, 1.4); sunLight.position.set(0, 0, 0); scene.add(sunLight);
const fill = new THREE.DirectionalLight(0x6aa8ff, 1.2); fill.position.set(-6, 4, 6); scene.add(fill);

function glowTexture() {
  const c = document.createElement('canvas'); c.width = c.height = 256;
  const g = c.getContext('2d'); const grad = g.createRadialGradient(128,128,4,128,128,128);
  grad.addColorStop(0, 'rgba(255,248,210,1)'); grad.addColorStop(0.18, 'rgba(255,207,106,.95)'); grad.addColorStop(0.48, 'rgba(255,150,60,.28)'); grad.addColorStop(1, 'rgba(255,120,30,0)');
  g.fillStyle = grad; g.fillRect(0,0,256,256);
  const t = new THREE.CanvasTexture(c); t.colorSpace = THREE.SRGBColorSpace; return t;
}
const sunUniforms = {time: {value: 0}};
const sunMaterial = new THREE.ShaderMaterial({
  uniforms: sunUniforms,
  vertexShader: `
    varying vec2 vUv;
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    void main() {
      vUv = uv;
      vWorldNormal = normalize(mat3(modelMatrix) * normal);
      vec4 world = modelMatrix * vec4(position, 1.0);
      vWorldPosition = world.xyz;
      gl_Position = projectionMatrix * viewMatrix * world;
    }
  `,
  fragmentShader: `
    uniform float time;
    varying vec2 vUv;
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
    float noise(vec2 p) {
      vec2 i = floor(p); vec2 f = fract(p);
      f = f * f * (3.0 - 2.0 * f);
      float a = hash(i);
      float b = hash(i + vec2(1.0, 0.0));
      float c = hash(i + vec2(0.0, 1.0));
      float d = hash(i + vec2(1.0, 1.0));
      return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
    }
    float fbm(vec2 p) {
      float v = 0.0; float a = 0.5;
      for (int i = 0; i < 5; i++) { v += a * noise(p); p *= 2.0; a *= 0.5; }
      return v;
    }
    void main() {
      vec3 V = normalize(cameraPosition - vWorldPosition);
      float mu = max(dot(normalize(vWorldNormal), V), 0.0);
      float limb = pow(mu, 0.42);
      float gran = fbm(vUv * 18.0 + vec2(time * 0.03, time * 0.01));
      vec3 core = mix(vec3(1.0, 0.22, 0.01), vec3(1.0, 0.82, 0.42), limb);
      core *= 0.82 + 0.38 * gran;
      core += vec3(1.0, 0.55, 0.12) * pow(limb, 3.0) * 0.25;
      core += (hash(gl_FragCoord.xy) - 0.5) / 255.0;
      gl_FragColor = vec4(core * 2.6, 1.0);
      #include <tonemapping_fragment>
      #include <colorspace_fragment>
    }
  `,
});
const sun = new THREE.Mesh(new THREE.SphereGeometry(1.1, 96, 64), sunMaterial);
scene.add(sun);
const sunGlow = new THREE.Sprite(new THREE.SpriteMaterial({map:glowTexture(), color:0xffcf7a, transparent:true, blending:THREE.AdditiveBlending, depthWrite:false}));
sunGlow.scale.set(9.0, 9.0, 1); scene.add(sunGlow);
const sunGlow2 = new THREE.Sprite(new THREE.SpriteMaterial({map:glowTexture(), color:0xff9a4a, transparent:true, opacity:0.28, blending:THREE.AdditiveBlending, depthWrite:false}));
sunGlow2.scale.set(13.5, 13.5, 1); scene.add(sunGlow2);

function streakTexture() {
  const c = document.createElement('canvas'); c.width = 1024; c.height = 64;
  const g = c.getContext('2d');
  const grad = g.createLinearGradient(0, 0, 1024, 0);
  grad.addColorStop(0, 'rgba(255,150,60,0)');
  grad.addColorStop(0.35, 'rgba(255,190,110,.35)');
  grad.addColorStop(0.5, 'rgba(255,245,220,1)');
  grad.addColorStop(0.65, 'rgba(255,190,110,.35)');
  grad.addColorStop(1, 'rgba(255,150,60,0)');
  g.fillStyle = grad; g.fillRect(0, 0, 1024, 64);
  const t = new THREE.CanvasTexture(c); t.colorSpace = THREE.SRGBColorSpace; return t;
}
const sunStreak = new THREE.Sprite(new THREE.SpriteMaterial({map:streakTexture(), color:0xffc27a, transparent:true, opacity:0.22, blending:THREE.AdditiveBlending, depthWrite:false}));
sunStreak.scale.set(15, 0.22, 1); scene.add(sunStreak);

const earthPivot = new THREE.Group(); scene.add(earthPivot);
const earthTilt = new THREE.Group(); earthPivot.add(earthTilt);
earthTilt.quaternion.setFromUnitVectors(new THREE.Vector3(0,1,0), AXIS);

const earthUniforms = {
  dayMap: {value: earthDayTex},
  nightMap: {value: earthNightTex},
  cloudMap: {value: earthCloudTex},
  specMap: {value: earthSpecTex},
  normalMap: {value: earthNormalTex},
  sunDirection: {value: new THREE.Vector3(1, 0, 0)},
  time: {value: 0},
};
const earthMaterial = new THREE.ShaderMaterial({
  uniforms: earthUniforms,
  vertexShader: `
    varying vec2 vUv;
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    void main() {
      vUv = uv;
      vWorldNormal = normalize(mat3(modelMatrix) * normal);
      vec4 world = modelMatrix * vec4(position, 1.0);
      vWorldPosition = world.xyz;
      gl_Position = projectionMatrix * viewMatrix * world;
    }
  `,
  fragmentShader: `
    uniform sampler2D dayMap;
    uniform sampler2D nightMap;
    uniform sampler2D cloudMap;
    uniform sampler2D specMap;
    uniform sampler2D normalMap;
    uniform vec3 sunDirection;
    uniform float time;
    varying vec2 vUv;
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }

    void main() {
      vec3 V = normalize(cameraPosition - vWorldPosition);
      vec3 N = normalize(vWorldNormal);
      vec3 dpdx = dFdx(vWorldPosition);
      vec3 dpdy = dFdy(vWorldPosition);
      vec2 duvdx = dFdx(vUv);
      vec2 duvdy = dFdy(vUv);
      vec3 T = normalize(dpdx * duvdy.y - dpdy * duvdx.y);
      vec3 B = normalize(-dpdx * duvdy.x + dpdy * duvdx.x);
      vec3 nTex = texture2D(normalMap, vUv).xyz * 2.0 - 1.0;
      vec3 mappedN = normalize(T * nTex.x + B * nTex.y + N * nTex.z);
      N = normalize(mix(N, mappedN, 0.38));
      float ndl = dot(N, sunDirection);
      float day = smoothstep(-0.16, 0.22, ndl);
      vec3 dayColor = texture2D(dayMap, vUv).rgb;
      dayColor = mix(vec3(dot(dayColor, vec3(0.2126, 0.7152, 0.0722))), dayColor, 1.18);
      vec3 nightColor = texture2D(nightMap, vUv).rgb * vec3(1.0, 0.78, 0.45) * 1.6;
      vec3 base = mix(nightColor * 0.55, dayColor, day);
      vec2 cloudUv = vUv + vec2(time * 0.0035, 0.0);
      float cloud = smoothstep(0.45, 0.94, texture2D(cloudMap, cloudUv).r);
      base = mix(base, vec3(0.94, 0.97, 1.0), cloud * 0.38 * day);
      base *= 1.0 - cloud * 0.10 * day;
      float specMask = texture2D(specMap, vUv).r;
      vec3 H = normalize(sunDirection + V);
      float spec = pow(max(dot(N, H), 0.0), 110.0) * specMask * day;
      base += vec3(1.0, 0.84, 0.58) * spec * 1.7;
      float fresnel = pow(1.0 - max(dot(N, V), 0.0), 3.0);
      base += vec3(0.16, 0.42, 0.95) * fresnel * (0.22 + 0.55 * day);
      base += vec3(1.0, 0.42, 0.10) * pow(max(ndl, 0.0), 12.0) * 0.10;
      base += (hash(gl_FragCoord.xy) - 0.5) / 255.0;
      gl_FragColor = vec4(base, 1.0);
      #include <tonemapping_fragment>
      #include <colorspace_fragment>
    }
  `,
});
const earth = new THREE.Mesh(new THREE.SphereGeometry(0.82, 128, 96), earthMaterial);
earthTilt.add(earth);

const atmosphereUniforms = {sunDirection: {value: new THREE.Vector3(1, 0, 0)}};
const atmosphereMaterial = new THREE.ShaderMaterial({
  uniforms: atmosphereUniforms,
  vertexShader: `
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    void main() {
      vWorldNormal = normalize(mat3(modelMatrix) * normal);
      vec4 world = modelMatrix * vec4(position, 1.0);
      vWorldPosition = world.xyz;
      gl_Position = projectionMatrix * viewMatrix * world;
    }
  `,
  fragmentShader: `
    uniform vec3 sunDirection;
    varying vec3 vWorldNormal;
    varying vec3 vWorldPosition;
    float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
    void main() {
      vec3 V = normalize(cameraPosition - vWorldPosition);
      vec3 N = normalize(vWorldNormal);
      float rim = pow(1.0 - abs(dot(N, V)), 2.2);
      float lit = clamp(dot(N, sunDirection) * 0.5 + 0.5, 0.0, 1.0);
      vec3 col = mix(vec3(0.05, 0.16, 0.45), vec3(0.35, 0.72, 1.0), lit);
      col += (hash(gl_FragCoord.xy) - 0.5) / 255.0;
      gl_FragColor = vec4(col * rim, rim * 0.55);
      #include <tonemapping_fragment>
      #include <colorspace_fragment>
    }
  `,
  transparent: true,
  blending: THREE.AdditiveBlending,
  depthWrite: false,
  side: THREE.BackSide,
});
const atmosphere = new THREE.Mesh(new THREE.SphereGeometry(0.94, 96, 72), atmosphereMaterial);
earthTilt.add(atmosphere);
const axisLine = new THREE.Line(new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(0,-1.25,0), new THREE.Vector3(0,1.25,0)]), new THREE.LineBasicMaterial({color:0xdff5ff, transparent:true, opacity:0.9, depthTest:false}));
axisLine.renderOrder = 4; earthTilt.add(axisLine);
const equator = new THREE.Mesh(new THREE.TorusGeometry(0.82, 0.012, 8, 96), new THREE.MeshBasicMaterial({color:0x7fdcff, transparent:true, opacity:0.55, depthTest:false}));
equator.rotation.x = Math.PI/2; equator.renderOrder = 3; earthTilt.add(equator);
const northRing = new THREE.Mesh(new THREE.TorusGeometry(0.70, 0.014, 8, 96), new THREE.MeshBasicMaterial({color:0xffd166, transparent:true, opacity:0.0, depthTest:false}));
northRing.rotation.x = Math.PI/2; northRing.position.y = 0.36; northRing.renderOrder = 5; earthTilt.add(northRing);
const southRing = new THREE.Mesh(new THREE.TorusGeometry(0.70, 0.014, 8, 96), new THREE.MeshBasicMaterial({color:0x63dcff, transparent:true, opacity:0.0, depthTest:false}));
southRing.rotation.x = Math.PI/2; southRing.position.y = -0.36; southRing.renderOrder = 5; earthTilt.add(southRing);

const orbitPoints = [];
for (let i=0;i<160;i++) { const a = i/160*Math.PI*2; orbitPoints.push(new THREE.Vector3(Math.cos(a)*5.5,0,Math.sin(a)*5.5)); }
const orbitLine = new THREE.LineLoop(new THREE.BufferGeometry().setFromPoints(orbitPoints), new THREE.LineBasicMaterial({color:0x3c7894, transparent:true, opacity:0.52}));
scene.add(orbitLine);

const rayGeo = new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(0,0,0), new THREE.Vector3(1,0,0)]);
const rayLine = new THREE.Line(rayGeo, new THREE.LineBasicMaterial({color:0xffd166, transparent:true, opacity:0.25})); scene.add(rayLine);
const rayCone = new THREE.Mesh(new THREE.ConeGeometry(0.09,0.28,12), new THREE.MeshBasicMaterial({color:0xffd166, transparent:true, opacity:0.9})); scene.add(rayCone);

const starCount = 1800;
const stars = new Float32Array(starCount * 3);
const starColors = new Float32Array(starCount * 3);
for (let i = 0; i < starCount; i++) {
  const r = 22 + hash(i + 1) * 30;
  const a = hash(i + 101) * Math.PI * 2;
  const b = (hash(i + 201) - 0.5) * Math.PI;
  stars[i * 3] = Math.cos(b) * Math.cos(a) * r;
  stars[i * 3 + 1] = Math.sin(b) * r;
  stars[i * 3 + 2] = Math.cos(b) * Math.sin(a) * r;
  const tint = hash(i + 301);
  starColors[i * 3] = 0.62 + tint * 0.38;
  starColors[i * 3 + 1] = 0.76 + hash(i + 401) * 0.24;
  starColors[i * 3 + 2] = 0.88 + hash(i + 501) * 0.12;
}
const starGeo = new THREE.BufferGeometry();
starGeo.setAttribute('position', new THREE.BufferAttribute(stars, 3));
starGeo.setAttribute('color', new THREE.BufferAttribute(starColors, 3));
const starField = new THREE.Points(starGeo, new THREE.PointsMaterial({size:0.045, vertexColors:true, transparent:true, opacity:0.9, depthWrite:false, sizeAttenuation:true}));
scene.add(starField);

function orbitAngle(t) {
  if (t < 12) return -0.55 + ease(t/6)*0.55 + 0.06*Math.sin(t*0.85)*(1-ease(t/6));
  if (t < 20) return 0.045*Math.sin((t-12)/8*Math.PI*2);
  if (t < 26) return Math.PI*ease((t-20)/6);
  return Math.PI;
}
function projectToScreen(v) { const p=v.clone().project(camera); return {x:(p.x*.5+.5)*W, y:(-p.y*.5+.5)*H, visible:p.z<1 && p.z>-1}; }
function pill(text, x, y, color=C.ink, bg='rgba(4,10,18,.78)', size=17, align='left') {
  ctx.save(); ctx.font = `600 ${size}px "Courier New",monospace`; const w=ctx.measureText(text).width+18; const h=size+14;
  const rawX=align==='center'?x-w/2:x; const bx=Math.max(12,Math.min(W-12-w,rawX)); const by=Math.max(h+8,Math.min(H-12,y));
  ctx.fillStyle=bg; ctx.strokeStyle='rgba(160,220,255,.18)'; ctx.lineWidth=1; ctx.beginPath(); ctx.roundRect(bx,by-h+4,w,h,7); ctx.fill(); ctx.stroke();
  ctx.fillStyle=color; ctx.textAlign='left'; ctx.fillText(text,bx+9,by); ctx.restore();
}
function drawHeader() {
  ctx.save(); ctx.fillStyle=C.muted; ctx.font='500 16px "Courier New",monospace'; ctx.fillText('HIMIND / CELESTIAL MOTION', 70, 92); ctx.textAlign='right'; ctx.fillStyle=C.cyan; ctx.fillText('SCHEMATIC / NOT TO SCALE', W-70, 92); ctx.textAlign='left'; ctx.strokeStyle='rgba(160,220,255,.18)'; ctx.lineWidth=1; ctx.beginPath(); ctx.moveTo(70,116); ctx.lineTo(W-70,116); ctx.stroke(); ctx.restore();
}
function drawIntroTitle(t) {
  const alpha=Math.min(1, t/0.5, (2.7-t)/0.45); if (alpha <= 0) return;
  ctx.save(); ctx.globalAlpha=clamp(alpha,0,1);
  ctx.fillStyle=C.ink; ctx.font='700 68px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, 74, 235);
  ctx.fillStyle=C.muted; ctx.font='500 25px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.subtitle, 78, 288);
  ctx.fillStyle=C.gold; ctx.font='600 16px "Courier New",monospace'; ctx.fillText('00 / 05', 78, 330);
  ctx.strokeStyle='rgba(255,209,102,.65)'; ctx.lineWidth=3; ctx.beginPath(); ctx.moveTo(78,354); ctx.lineTo(78+420*ease(t/0.8),354); ctx.stroke();
  ctx.restore();
}
function drawSceneText(t) {
  const window=sceneAt(t); const text=sceneText(window.id);
  if (t < 2.7) return;
  const local=window.id==='orbit'?Math.max(0,t-2.7):t-window.start; const alpha=Math.min(1, (local+0.12)/0.7, (window.end-t)/0.45);
  if (alpha <= 0) return;
  ctx.save(); ctx.globalAlpha=clamp(alpha,0,1); ctx.fillStyle=C.ink; ctx.font='700 68px "Microsoft YaHei", sans-serif'; ctx.fillText(text.title, 74, 235); ctx.fillStyle=C.muted; ctx.font='500 25px "Microsoft YaHei", sans-serif'; ctx.fillText(text.caption, 78, 288); ctx.fillStyle=C.cyan; ctx.font='600 16px "Courier New",monospace'; ctx.fillText(`0${SCENE_WINDOWS.indexOf(window)+1} / 05`, 78, 330); ctx.strokeStyle='rgba(99,220,255,.65)'; ctx.lineWidth=3; ctx.beginPath(); ctx.moveTo(78,354); ctx.lineTo(78+420*ease(local/0.8),354); ctx.stroke(); ctx.restore();
}
function drawDistanceNote(t) {
  const p = Math.min(ease(t/0.8), 1-ease((t-5.0)/1.0)); if (p <= 0) return;
  ctx.save(); ctx.globalAlpha=p; ctx.fillStyle='rgba(255,177,92,.10)'; ctx.strokeStyle='rgba(255,177,92,.42)'; ctx.lineWidth=2; ctx.beginPath(); ctx.roundRect(74,1180,430,86,10); ctx.fill(); ctx.stroke(); ctx.fillStyle=C.gold; ctx.font='600 17px "Courier New",monospace'; ctx.fillText('KEY POINT', 94, 1210); ctx.fillStyle=C.ink; ctx.font='700 26px "Microsoft YaHei", sans-serif'; ctx.fillText('距离 ≠ 季节', 94, 1246); ctx.restore();
}
function drawFacts(t) {
  const p=ease((t-4.8)/0.8); if(p<=0) return;
  ctx.save(); ctx.globalAlpha=p; DATA.facts.forEach((fact,i)=>{ const x=74+i*310; ctx.fillStyle='rgba(8,18,30,.72)'; ctx.strokeStyle='rgba(160,220,255,.16)'; ctx.lineWidth=1; ctx.beginPath(); ctx.roundRect(x,1450,280,92,10); ctx.fill(); ctx.stroke(); ctx.fillStyle=i===0?C.gold:i===1?C.green:C.cyan; ctx.font='700 24px "Courier New",monospace'; ctx.fillText(fact.value,x+16,1488); ctx.fillStyle=C.muted; ctx.font='500 18px "Microsoft YaHei", sans-serif'; ctx.fillText(fact.label,x+16,1520); }); ctx.restore();
}
function drawConclusion(t) {
  const p=ease((t-26.2)/0.7); if(p<=0) return;
  ctx.save(); ctx.globalAlpha=p; ctx.textAlign='center'; ctx.fillStyle=C.ink; ctx.font='700 54px "Microsoft YaHei", sans-serif'; ctx.fillText('四季 = 地轴倾斜 + 公转', W/2, 1650); ctx.fillStyle=C.gold; ctx.font='600 24px "Courier New",monospace'; ctx.fillText('NOT DISTANCE', W/2, 1694); ctx.strokeStyle='rgba(255,209,102,.65)'; ctx.lineWidth=3; ctx.beginPath(); ctx.moveTo(W/2-190,1724); ctx.lineTo(W/2+190,1724); ctx.stroke(); ctx.restore();
}
function drawLabels(zoom, sceneId, earthPos) {
  const sunP=projectToScreen(new THREE.Vector3(0,0,0)); const earthP=projectToScreen(earthPos);
  if (zoom < 0.55 && sunP.visible) pill('太阳', sunP.x+34, sunP.y-12, C.gold);
  if (earthP.visible) pill('地球', earthP.x+36, earthP.y-8, C.cyan);
  if (zoom > 0.55) {
    const north=projectToScreen(earthTilt.localToWorld(new THREE.Vector3(0,1.25,0)));
    if (north.visible) pill('N', north.x+18, north.y-8, C.ink, 'rgba(4,10,18,.88)', 16);
    const axisMid=projectToScreen(earthTilt.localToWorld(new THREE.Vector3(0,0.2,0)));
    if (axisMid.visible) pill('地轴 23.44°', axisMid.x+40, axisMid.y, C.gold, 'rgba(4,10,18,.82)', 15);
    if (sceneId === 'north' || sceneId === 'south') {
      const rayMid=projectToScreen(earthPos.clone().multiplyScalar(0.52));
      if (rayMid.visible) pill('阳光', rayMid.x, rayMid.y-18, C.gold, 'rgba(4,10,18,.82)', 15, 'center');
    }
  }
}
function drawProgress(t) {
  const p=clamp(t/DURATION,0,1); ctx.save(); ctx.fillStyle='rgba(238,248,255,.12)'; ctx.fillRect(70,1845,W-140,4); ctx.fillStyle=C.cyan; ctx.fillRect(70,1845,(W-140)*p,4); SCENE_WINDOWS.forEach((s,i)=>{const x=70+(W-140)*(s.start/DURATION);ctx.fillStyle=t>=s.start?C.cyan:'rgba(238,248,255,.22)';ctx.beginPath();ctx.arc(x,1847,3,0,Math.PI*2);ctx.fill();}); ctx.fillStyle=C.muted; ctx.font='500 15px "Courier New",monospace'; ctx.fillText(DATA.source,70,1815); ctx.textAlign='right'; ctx.fillText('30s / 1080×1920',W-70,1815); ctx.restore();
}
function drawOverlay(t, zoom, sceneId, earthPos) {
  ctx.clearRect(0,0,W,H);
  drawHeader(); drawIntroTitle(t); drawSceneText(t); drawDistanceNote(t); drawFacts(t); drawLabels(zoom,sceneId,earthPos); drawConclusion(t); drawProgress(t);
}
function renderAt(t) {
  const time=clamp(t,0,DURATION); const angle=orbitAngle(time); const earthPos=new THREE.Vector3(Math.cos(angle)*5.5,0,Math.sin(angle)*5.5);
  earthPivot.position.copy(earthPos); earth.rotation.y=time*(0.38+0.06*Math.sin(time*0.17)); axisLine.material.opacity=0.78+0.14*Math.sin(time*1.1); earthUniforms.sunDirection.value.copy(earthPos).multiplyScalar(-1).normalize(); earthUniforms.time.value=time; atmosphereUniforms.sunDirection.value.copy(earthUniforms.sunDirection.value); sunUniforms.time.value=time; nebulaUniforms.time.value=time;
  const zoom=ease((time-4.0)/2.5)*(1-ease((time-25.0)/2.5));
  const intro=ease(time/4.0); const finale=ease((time-26.0)/4.0);
  const widePos=new THREE.Vector3(Math.sin(time*0.45)*1.6*intro, 7.2-0.5*ease(time/6)+Math.sin(time*0.34)*0.35*intro-0.35*finale, 15.4-3.4*ease(time/6)+1.1*finale);
  const sweep=0.16*Math.sin((time-6.0)/6.0*Math.PI);
  const closePos=earthPos.clone().add(new THREE.Vector3(3.5*Math.cos(sweep)-5.6*Math.sin(sweep),2.5,3.5*Math.sin(sweep)+5.6*Math.cos(sweep)));
  camera.position.lerpVectors(widePos,closePos,zoom); const target=new THREE.Vector3(0,0,0).lerp(earthPos,zoom); camera.lookAt(target);
  starField.rotation.y=time*0.012; starField.rotation.x=Math.sin(time*0.05)*0.018; const glowScale=9.0+0.22*Math.sin(time*0.55); const glowScale2=13.5+0.3*Math.sin(time*0.41+1.2); sunGlow.scale.set(glowScale,glowScale,1); sunGlow2.scale.set(glowScale2,glowScale2,1);
  const sceneId=sceneAt(time).id;
  northRing.material.opacity = sceneId==='north'?0.95:sceneId==='south'?0.12:0.30;
  southRing.material.opacity = sceneId==='south'?0.95:sceneId==='north'?0.12:0.30;
  const rayDir=earthPos.clone().normalize(); const start=rayDir.clone().multiplyScalar(1.25); const end=earthPos.clone().sub(rayDir.clone().multiplyScalar(1.02));
  rayGeo.setFromPoints([start,end]); rayGeo.attributes.position.needsUpdate=true; rayCone.position.copy(end); rayCone.quaternion.setFromUnitVectors(new THREE.Vector3(0,1,0),rayDir);
  const rayOpacity=sceneId==='north'||sceneId==='south'?0.9:sceneId==='tilt'?0.25:0.16; rayLine.material.opacity=rayOpacity; rayCone.material.opacity=rayOpacity;
  renderer.render(scene,camera); drawOverlay(time,zoom,sceneId,earthPos);
}
window.__AUDIT_META__ = (t) => {
  const window = sceneAt(t);
  return {
    title: t < 2.7 ? DATA.title : sceneText(window.id).title,
    caption: t < 2.7 ? DATA.subtitle : sceneText(window.id).caption,
  };
};
window.renderAt=renderAt; window.__DURATION__=DURATION; window.__FPS__=FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__=true; }));

