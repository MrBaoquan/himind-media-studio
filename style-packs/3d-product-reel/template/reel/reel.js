import * as THREE from '../node_modules/three/build/three.module.js';

const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const webglCanvas = document.getElementById('webgl');
const overlay = document.getElementById('overlay');
const ctx = overlay.getContext('2d');
const DEFAULT = {
  title: '为创作者而生',
  product: 'HIMIND ONE',
  subtitle: 'PRODUCT FILM / 2026',
  features: ['金属一体成型', '全天候创作', '端侧智能'],
  price: '¥ 4,999',
  source: 'HIMIND / 3D PRODUCT REEL',
};
let DATA = DEFAULT;
async function loadData() {
  try {
    const response = await fetch('/input/data.json', {cache: 'no-store'});
    if (response.ok) {
      const value = await response.json();
      const features = Array.isArray(value.features) ? value.features.slice(0, 3).map(String) : DEFAULT.features;
      DATA = {title:String(value.title || DEFAULT.title), product:String(value.product || DEFAULT.product), subtitle:String(value.subtitle || DEFAULT.subtitle), features:features.length ? features : DEFAULT.features, price:String(value.price || DEFAULT.price), source:String(value.source || DEFAULT.source)};
    }
  } catch { DATA = DEFAULT; }
}
const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
const ease = (v) => 1 - Math.pow(1 - clamp(v, 0, 1), 3);
const hash = (n) => { const x = Math.sin(n * 127.1 + 311.7) * 43758.5453; return x - Math.floor(x); };
const C = {ink:'#f3f9ff',muted:'rgba(243,249,255,.58)',dim:'rgba(243,249,255,.28)',cyan:'#6ce7ff',pink:'#ff5fa2',gold:'#f4c96b'};

const renderer = new THREE.WebGLRenderer({canvas: webglCanvas, antialias: true, preserveDrawingBuffer: true, powerPreference: 'high-performance'});
renderer.setPixelRatio(1);
renderer.setSize(W, H, false);
renderer.outputColorSpace = THREE.SRGBColorSpace;
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 1.15;
renderer.shadowMap.enabled = true;

const scene = new THREE.Scene();
scene.background = new THREE.Color(0x05070d);
scene.fog = new THREE.Fog(0x05070d, 10, 24);
const camera = new THREE.PerspectiveCamera(32, W / H, 0.1, 100);
camera.position.set(0, 2.2, 9.2);
camera.lookAt(0, 0, 0);

scene.add(new THREE.AmbientLight(0x30465a, 2.0));
const key = new THREE.DirectionalLight(0xffffff, 3.2); key.position.set(4, 7, 6); key.castShadow = true; scene.add(key);
const rimPink = new THREE.PointLight(0xff4f9d, 42, 20, 2); rimPink.position.set(-4.5, 2.5, 4); scene.add(rimPink);
const rimCyan = new THREE.PointLight(0x4de2ff, 38, 20, 2); rimCyan.position.set(4.5, -0.6, 4); scene.add(rimCyan);

const group = new THREE.Group();
scene.add(group);
const bodyMat = new THREE.MeshPhysicalMaterial({color:0xd6dee8, metalness:0.92, roughness:0.16, clearcoat:1, clearcoatRoughness:0.1});
const body = new THREE.Mesh(new THREE.BoxGeometry(2.5, 3.5, 0.38, 8, 8, 2), bodyMat);
body.position.y = 0.2; body.castShadow = true; body.receiveShadow = true; group.add(body);
const screenMat = new THREE.MeshBasicMaterial({color:0x0b2239});
const screen = new THREE.Mesh(new THREE.PlaneGeometry(2.06, 2.92), screenMat); screen.position.set(0, 0.2, 0.201); group.add(screen);
const glowPlane = new THREE.Mesh(new THREE.PlaneGeometry(2.06, 2.92), new THREE.MeshBasicMaterial({color:0x1d6d93, transparent:true, opacity:0.20, blending:THREE.AdditiveBlending})); glowPlane.position.set(0, 0.2, 0.205); group.add(glowPlane);
const edges = new THREE.LineSegments(new THREE.EdgesGeometry(body.geometry), new THREE.LineBasicMaterial({color:0x6ce7ff, transparent:true, opacity:0.85})); edges.position.copy(body.position); group.add(edges);
const ring = new THREE.Mesh(new THREE.TorusGeometry(2.35, 0.028, 12, 120), new THREE.MeshBasicMaterial({color:0xff5fa2, transparent:true, opacity:0.8})); ring.rotation.x = Math.PI / 2.4; ring.rotation.z = -0.2; group.add(ring);
const ring2 = new THREE.Mesh(new THREE.TorusGeometry(2.9, 0.018, 10, 120), new THREE.MeshBasicMaterial({color:0x6ce7ff, transparent:true, opacity:0.5})); ring2.rotation.x = Math.PI / 2.1; group.add(ring2);

const base = new THREE.Mesh(new THREE.CylinderGeometry(3.0, 3.4, 0.16, 64), new THREE.MeshPhysicalMaterial({color:0x111923, metalness:0.8, roughness:0.28})); base.position.y = -2.15; base.receiveShadow = true; scene.add(base);
const grid = new THREE.GridHelper(10, 20, 0x1a3345, 0x0d1b28); grid.position.y = -2.06; scene.add(grid);

const cubes = [];
for (let i = 0; i < 3; i++) {
  const color = [0xff5fa2, 0x6ce7ff, 0xf4c96b][i];
  const cube = new THREE.Mesh(new THREE.BoxGeometry(0.22, 0.22, 0.22), new THREE.MeshBasicMaterial({color, transparent:true, opacity:0.85}));
  cube.position.set(-2.7 + i * 2.7, 0.2 + (i - 1) * 0.9, 1.4); group.add(cube); cubes.push(cube);
}
const positions = new Float32Array(240 * 3);
for (let i = 0; i < 240; i++) { positions[i*3] = (hash(i+1)-.5)*14; positions[i*3+1] = (hash(i+101)-.5)*12; positions[i*3+2] = (hash(i+201)-.5)*12; }
const particleGeo = new THREE.BufferGeometry(); particleGeo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
const particles = new THREE.Points(particleGeo, new THREE.PointsMaterial({color:0x6ce7ff, size:0.028, transparent:true, opacity:0.45})); scene.add(particles);

function drawOverlay(t) {
  ctx.clearRect(0, 0, W, H);
  const titleP = ease(t / 1.2);
  ctx.save(); ctx.globalAlpha = titleP;
  ctx.fillStyle = C.ink; ctx.font = '700 74px "Microsoft YaHei", sans-serif'; ctx.fillText(DATA.title, 80, 250);
  ctx.fillStyle = C.cyan; ctx.font = '600 22px "Courier New",monospace'; ctx.fillText(DATA.subtitle, 84, 300);
  ctx.strokeStyle = 'rgba(108,231,255,.55)'; ctx.lineWidth = 3; ctx.beginPath(); ctx.moveTo(84, 336); ctx.lineTo(84 + 360 * titleP, 336); ctx.stroke(); ctx.restore();
  ctx.save(); ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText('HIMIND / PRODUCT FILM', 80, 104); ctx.textAlign = 'right'; ctx.fillText('STUDIO 01', W - 80, 104); ctx.textAlign = 'left'; ctx.restore();
  ctx.save(); ctx.fillStyle = C.ink; ctx.font = '700 30px "Courier New",monospace'; ctx.fillText(DATA.product, 84, 1230); ctx.restore();
  DATA.features.forEach((feature, i) => {
    const p = ease((t - (2.1 + i * 0.7)) / 0.7); if (p <= 0) return;
    const y = 1370 + i * 78; ctx.save(); ctx.globalAlpha = p; ctx.fillStyle = i === 1 ? C.pink : C.cyan; ctx.font = '700 18px "Courier New",monospace'; ctx.fillText(`0${i+1}`, 84, y); ctx.strokeStyle = 'rgba(243,249,255,.18)'; ctx.beginPath(); ctx.moveTo(128, y - 8); ctx.lineTo(560, y - 8); ctx.stroke(); ctx.fillStyle = C.ink; ctx.font = '700 34px "Microsoft YaHei", sans-serif'; ctx.fillText(feature, 146, y + 2); ctx.restore();
  });
  const priceP = ease((t - 8.0) / 0.7); if (priceP > 0) { ctx.save(); ctx.globalAlpha = priceP; ctx.textAlign = 'right'; ctx.fillStyle = C.gold; ctx.font = '700 42px "Courier New",monospace'; ctx.fillText(DATA.price, W - 84, 1700); ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText('AVAILABLE NOW', W - 84, 1734); ctx.restore(); }
  ctx.save(); ctx.fillStyle = C.muted; ctx.font = '500 17px "Courier New",monospace'; ctx.fillText(DATA.source, 84, 1815); ctx.fillStyle = C.cyan; ctx.fillRect(84, 1780, 180, 4); ctx.restore();
}
function renderAt(t) {
  const time = clamp(t, 0, DURATION);
  group.rotation.y = 0.16 + time * 0.36;
  group.position.y = Math.sin(time * 1.35) * 0.08;
  group.position.x = Math.sin(time * 0.45) * 0.06;
  camera.position.x = Math.sin(time * 0.28) * 0.25;
  camera.position.y = 2.2 + Math.sin(time * 0.35) * 0.08;
  camera.lookAt(0, 0, 0);
  ring.rotation.z = time * 0.22;
  ring2.rotation.z = -time * 0.16;
  cubes.forEach((cube, i) => { cube.position.y = 0.2 + (i - 1) * 0.9 + Math.sin(time * 1.4 + i) * 0.16; cube.rotation.x = time * 0.8 + i; cube.rotation.y = time * 0.6 + i; });
  particles.rotation.y = time * 0.02;
  renderer.render(scene, camera);
  drawOverlay(time);
}
window.renderAt = renderAt; window.__DURATION__ = DURATION; window.__FPS__ = FPS;
loadData().then(() => document.fonts.ready.then(() => { renderAt(0); window.__READY__ = true; }));
