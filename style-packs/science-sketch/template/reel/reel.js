import rough from './vendor/rough.esm.js';

const W = 1080, H = 1920, FPS = 30, DURATION = 45;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const rawRc = rough.canvas(canvas);
let boil = 0;
const rc = new Proxy(rawRc, {
  get(target, prop) {
    const fn = target[prop];
    if (typeof fn !== 'function') return fn;
    return (...args) => {
      const last = args[args.length - 1];
      if (last && typeof last === 'object' && !Array.isArray(last)) {
        args[args.length - 1] = {...last, seed: (Number(last.seed) || 1) + boil};
      }
      return fn.apply(target, args);
    };
  },
});
let NOW = 0;

const DEFAULT = {
  title: '天空为什么是蓝色的？',
  subtitle: '从阳光、空气分子到日落',
  source: '来源：NASA Space Place / NOAA JetStream',
  sources: [
    {id:'s1', publisher:'NASA Space Place', title:'Why Is the Sky Blue?', url:'https://spaceplace.nasa.gov/blue-sky/en/'},
    {id:'s2', publisher:'NOAA JetStream', title:'The Atmosphere', url:'https://www.noaa.gov/jetstream/atmosphere'},
  ],
  scenes: [
    {id:'hook', type:'title', start:0, end:5, kicker:'SCIENCE SKETCH 01', title:'天空为什么是蓝色的？', caption:'天空明明是空的，为什么抬头却是蓝色？', note:'先别急着说“反射海洋”'},
    {id:'spectrum', type:'spectrum', start:5, end:12, kicker:'01 / SUNLIGHT', title:'阳光不是只有白色', caption:'白光里藏着从红到紫的可见光谱。', labels:['红','橙','黄','绿','蓝','靛','紫']},
    {id:'scatter', type:'scatter', start:12, end:23, kicker:'02 / SCATTERING', title:'空气分子把蓝光散开', caption:'短波长的蓝光更容易被空气分子改变方向。', note:'瑞利散射 ≈ 1/λ⁴', labels:['太阳光','空气分子','蓝光被散射']},
    {id:'observer', type:'observer', start:23, end:31, kicker:'03 / YOUR EYES', title:'所以整片天空都在发蓝', caption:'蓝光从天空各处进入你的眼睛，而不只是来自太阳。', note:'紫光散射更强，但人眼对蓝更敏感'},
    {id:'sunset', type:'sunset', start:31, end:40, kicker:'04 / SUNSET', title:'日落时，蓝光先被散掉', caption:'光线穿过更厚的大气层，剩下的红橙色抵达眼睛。', note:'路径更长 → 蓝光被散掉更多'},
    {id:'takeaway', type:'takeaway', start:40, end:45, kicker:'TAKEAWAY', title:'你看到的蓝，是空气散射后的阳光。', caption:'不是天空被涂成蓝色，也不是海洋的反射。', source:'来源：NASA Space Place / NOAA JetStream'},
  ],
};

let DATA = DEFAULT;
const C = {paper:'#f7f4ea', paper2:'#fffdf7', ink:'#26313b', soft:'#65727a', faint:'rgba(38,49,59,.12)', blue:'#2b74b5', cyan:'#59b9d1', orange:'#d47c26', red:'#c54a3d', green:'#2d8a68', yellow:'#e6bd3e', violet:'#7652a7'};

function normalizeData(value) {
  if (!value || typeof value !== 'object') return DEFAULT;
  const scenes = Array.isArray(value.scenes) ? value.scenes.filter((s) => s && typeof s === 'object').map((s, i) => ({
    id:String(s.id || `scene-${i+1}`), type:String(s.type || 'title'), start:Number(s.start ?? i*7), end:Number(s.end ?? (i+1)*7),
    kicker:String(s.kicker || ''), title:String(s.title || ''), caption:String(s.caption || ''), note:String(s.note || ''),
    labels:Array.isArray(s.labels) ? s.labels.map(String) : [], source:String(s.source || ''),
  })) : DEFAULT.scenes;
  const sources = Array.isArray(value.sources) ? value.sources.filter((x) => x && typeof x === 'object').map((x) => ({
    id:String(x.id || ''), publisher:String(x.publisher || ''), title:String(x.title || ''), url:String(x.url || ''),
  })) : [];
  return {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), source:String(value.source || DEFAULT.source), sources:sources.length?sources:DEFAULT.sources, scenes:scenes.length?scenes:DEFAULT.scenes};
}
async function loadData(){try{const r=await fetch('/input/data.json',{cache:'no-store'});if(r.ok)DATA=normalizeData(await r.json());}catch{DATA=DEFAULT;}}
const clamp=(v,a,b)=>Math.max(a,Math.min(b,v));
const ease=(v)=>1-Math.pow(1-clamp(v,0,1),3);
const hash=(n)=>{const x=Math.sin(n*127.1+311.7)*43758.5453;return x-Math.floor(x);};
function setFont(size,weight=400,family='body'){
  const stack=family==='hand'?'"KaiTi","STKaiti","Segoe Print","Comic Sans MS","Noto Sans SC",sans-serif':family==='mono'?'"Cascadia Mono","JetBrains Mono",Consolas,monospace':'"Microsoft YaHei","Noto Sans SC","PingFang SC",sans-serif';
  ctx.font=`${weight} ${size}px ${stack}`;
}
function text(value,x,y,o={}){const{size=28,weight=400,family='body',color=C.ink,align='left',alpha=1,baseline='alphabetic'}=o;ctx.save();ctx.globalAlpha*=alpha;ctx.fillStyle=color;ctx.textAlign=align;ctx.textBaseline=baseline;setFont(size,weight,family);ctx.fillText(String(value),x,y);ctx.restore();}
function wrap(value,maxWidth,size,weight=400,family='body'){setFont(size,weight,family);const chars=[...String(value||'')];const lines=[];let line='';for(const ch of chars){const test=line+ch;if(line&&ctx.measureText(test).width>maxWidth){lines.push(line);line=ch;}else line=test;}if(line)lines.push(line);return lines;}
function sceneAt(t){for(const s of DATA.scenes){if(t>=s.start&&t<s.end)return s;}return DATA.scenes[DATA.scenes.length-1];}
function sceneProgress(scene,t){return clamp((t-scene.start)/Math.max(.001,scene.end-scene.start),0,1);}
function paper(t){ctx.fillStyle=C.paper;ctx.fillRect(0,0,W,H);const gx=W*.72+Math.sin(t*.33)*14,gy=H*.18+Math.cos(t*.27)*10;const g=ctx.createRadialGradient(gx,gy,20,gx,gy,W*.9);g.addColorStop(0,'rgba(174,207,221,.22)');g.addColorStop(.55,'rgba(221,214,190,.09)');g.addColorStop(1,'rgba(255,255,255,0)');ctx.fillStyle=g;ctx.fillRect(0,0,W,H);ctx.save();ctx.strokeStyle='rgba(38,49,59,.025)';ctx.lineWidth=1;const dx=Math.sin(t*.42)*3,dy=Math.cos(t*.35)*2;for(let y=80;y<H;y+=42){ctx.beginPath();ctx.moveTo(70,y+dy*.4);ctx.lineTo(W-70,y+dy*.4);ctx.stroke();}for(let i=0;i<240;i++){const x=hash(i+1)*W+dx,y=hash(i+101)*H+dy;ctx.fillStyle=`rgba(38,49,59,${(.012+hash(i+201)*.018).toFixed(3)})`;ctx.fillRect(x,y,2,2);}ctx.restore();}
function frame(){rc.rectangle(48,48,W-96,H-96,{stroke:'rgba(38,49,59,.46)',strokeWidth:3,roughness:1.35,bowing:1.15,seed:11});rc.line(92,132,W-92,132,{stroke:'rgba(38,49,59,.25)',strokeWidth:2,roughness:1.1,seed:12});}
function header(scene,t){const p=ease((t-scene.start)/.65);text('HIMIND / SCIENCE SKETCH',92,112,{size:21,weight:700,family:'mono',color:C.ink,alpha:p});text(scene.kicker||'SCIENCE SKETCH',W-92,112,{size:20,weight:700,family:'mono',color:C.blue,align:'right',alpha:p});rc.line(92,132,W-92,132,{stroke:'rgba(38,49,59,.25)',strokeWidth:2,roughness:1.1,seed:12});}
function titleBlock(scene,t){const p=ease((t-scene.start)/.7);if(p<=0)return;const size=[...String(scene.title)].length>14?46:58;text(scene.title,W/2,265,{size,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});const lines=wrap(scene.caption,W-220,28,400,'body');lines.forEach((line,i)=>text(line,W/2,330+i*40,{size:28,weight:400,color:C.soft,align:'center',alpha:p}));if(scene.note)text(scene.note,W/2,430,{size:21,weight:600,family:'mono',color:C.orange,align:'center',alpha:p});}
function footer(scene,t){const src=scene.source||DATA.source;if(src)text(src,W/2,1778,{size:21,weight:400,color:C.soft,align:'center',alpha:ease((t-scene.start)/.8)});text('SCIENCE / HAND-DRAWN / 01',92,1814,{size:19,weight:600,family:'mono',color:C.soft,alpha:.8});}
function sun(cx,cy,r,alpha=1){ctx.save();ctx.globalAlpha*=alpha;const R=r*(1+Math.sin(NOW*1.9)*.018);const g=ctx.createRadialGradient(cx,cy,R*.2,cx,cy,R);g.addColorStop(0,'#ffe58a');g.addColorStop(.55,'#f4ba4d');g.addColorStop(1,'rgba(244,186,77,.08)');ctx.fillStyle=g;ctx.beginPath();ctx.arc(cx,cy,R,0,Math.PI*2);ctx.fill();rc.circle(cx,cy,R*.78,{stroke:C.orange,strokeWidth:3.2,roughness:1.3,seed:41});for(let i=0;i<8;i++){const a=i*Math.PI/4+Math.sin(NOW*.5)*.015;rc.line(cx+Math.cos(a)*R*1.08,cy+Math.sin(a)*R*1.08,cx+Math.cos(a)*R*1.28,cy+Math.sin(a)*R*1.28,{stroke:C.orange,strokeWidth:3,roughness:1.2,seed:42+i});}ctx.restore();}
function molecule(cx,cy,r,color=C.blue,seed=1){const mx=cx+Math.sin(NOW*.9+seed)*2,my=cy+Math.cos(NOW*.7+seed)*2;rc.circle(mx,my,r,{stroke:color,strokeWidth:2.8,roughness:1.4,seed});rc.circle(mx-r*.38,my-r*.25,r*.18,{stroke:color,strokeWidth:2,roughness:1.1,seed:seed+1});rc.circle(mx+r*.4,my+r*.25,r*.18,{stroke:color,strokeWidth:2,roughness:1.1,seed:seed+2});}
function arrow(x1,y1,x2,y2,color=C.blue,seed=1,width=3){rc.line(x1,y1,x2,y2,{stroke:color,strokeWidth:width,roughness:1.2,seed});const a=Math.atan2(y2-y1,x2-x1);rc.line(x2,y2,x2-Math.cos(a-.45)*16,y2-Math.sin(a-.45)*16,{stroke:color,strokeWidth:width,roughness:1,seed:seed+1});rc.line(x2,y2,x2-Math.cos(a+.45)*16,y2-Math.sin(a+.45)*16,{stroke:color,strokeWidth:width,roughness:1,seed:seed+2});}
function sketchPanel(x,y,w,h,seed=1,color='rgba(43,116,181,.18)'){ctx.save();ctx.fillStyle='rgba(255,253,247,.62)';ctx.fillRect(x,y,w,h);rc.rectangle(x,y,w,h,{stroke:color,strokeWidth:2.4,roughness:1.45,bowing:1.2,seed});ctx.restore();}
function drawTitleScene(scene,t,p){sun(760,760,100,p);rc.line(520,720,690,810,{stroke:C.orange,strokeWidth:2.5,roughness:1.3,seed:51});text('？',520+Math.sin(t*1.1)*3,730+Math.cos(t*1.3)*4,{size:120,weight:700,family:'hand',color:C.blue,align:'center',alpha:p});sketchPanel(240,920,600,210,60);text('先问一个问题，再找机制',W/2,1000,{size:30,weight:600,family:'hand',color:C.ink,align:'center',alpha:p});text('从阳光、空气分子到你的眼睛',W/2,1060,{size:24,weight:400,color:C.soft,align:'center',alpha:p});}
function spectrumColors(){return ['#c54a3d','#d47c26','#e6bd3e','#2d8a68','#2b74b5','#7652a7','#7652a7'];}
function drawSpectrum(scene,t,p){const cols=spectrumColors();const labels=scene.labels.length>=7?scene.labels:['红','橙','黄','绿','蓝','靛','紫'];rc.line(140,1000,410,820,{stroke:C.ink,strokeWidth:3,roughness:1.2,seed:71});rc.line(410,820,680,1000,{stroke:C.ink,strokeWidth:3,roughness:1.2,seed:72});rc.line(140,1000,680,1000,{stroke:C.ink,strokeWidth:3,roughness:1.2,seed:73});ctx.save();ctx.beginPath();ctx.moveTo(410,830);ctx.lineTo(670,990);ctx.lineTo(150,990);ctx.closePath();ctx.clip();const g=ctx.createLinearGradient(150,0,670,0);cols.forEach((c,i)=>g.addColorStop(i/(cols.length-1),c));ctx.globalAlpha=p*(.88+Math.sin(t*1.7)*.04);ctx.fillStyle=g;ctx.fillRect(150,820,520,180);ctx.restore();for(let i=0;i<cols.length;i++){const x=210+i*70;rc.line(650,1000,x,1270,{stroke:cols[i],strokeWidth:6+Math.sin(t*2.2+i)*0.6,roughness:1.1,seed:80+i});text(labels[i],x,1320,{size:24,weight:700,family:'hand',color:cols[i],align:'center',alpha:p});}text('白光',160,780,{size:26,weight:700,family:'hand',color:C.ink,alpha:p});text('棱镜',420,780,{size:26,weight:700,family:'hand',color:C.blue,alpha:p});}
function drawScatter(scene,t,p){sun(240,760,95,p);molecule(590,720,24,C.blue,101);molecule(720,680,18,C.green,111);molecule(780,820,22,C.violet,121);for(let i=0;i<9;i++){const a=-.8+i*.2;arrow(560,760,560+Math.cos(a)*240,760+Math.sin(a)*240,i%2?C.cyan:C.blue,130+i,2.4);}text('太阳光进入大气',170,1070,{size:24,weight:600,family:'hand',color:C.ink,alpha:p});text('空气分子',660,1040,{size:24,weight:700,family:'hand',color:C.blue,align:'center',alpha:p});text('蓝光被散射到四面八方',760,1120,{size:23,weight:600,family:'hand',color:C.cyan,align:'center',alpha:p});sketchPanel(220,1220,640,150,150);text('短波长 → 更容易被散射',W/2,1300,{size:25,weight:600,family:'hand',color:C.ink,align:'center',alpha:p});}
function drawObserver(scene,t,p){ctx.save();ctx.fillStyle='rgba(117,192,221,.42)';ctx.fillRect(150,560,780,520);const g=ctx.createLinearGradient(0,560,0,1080);g.addColorStop(0,'rgba(85,170,214,.55)');g.addColorStop(1,'rgba(235,246,249,.22)');ctx.fillStyle=g;ctx.fillRect(150,560,780,520);ctx.restore();rc.rectangle(150,560,780,520,{stroke:'rgba(43,116,181,.35)',strokeWidth:2.5,roughness:1.2,seed:171});for(let i=0;i<7;i++){const x=220+i*110;arrow(x,620,420+(i%3)*80+Math.cos(t*1.1+i)*6,1040+Math.sin(t*1.4+i)*8,C.blue,180+i,2.2);}text('来自天空各处的蓝光',W/2,1160,{size:25,weight:600,family:'hand',color:C.blue,align:'center',alpha:p});sketchPanel(300,1240,480,170,190);rc.ellipse(540,1325,160,62,{stroke:C.ink,strokeWidth:3,roughness:1.2,seed:191});rc.circle(540,1325,34,{stroke:C.blue,strokeWidth:3,roughness:1.1,seed:192});text('眼睛',540,1438,{size:24,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});}
function drawSunset(scene,t,p){const left=90,right=560,half=430;sketchPanel(left,560,half,620,210,'rgba(43,116,181,.25)');sketchPanel(right,560,half,620,220,'rgba(197,74,61,.3)');const day=ctx.createLinearGradient(0,560,0,1180);day.addColorStop(0,'rgba(110,193,226,.50)');day.addColorStop(1,'rgba(246,249,237,.25)');ctx.fillStyle=day;ctx.fillRect(left,560,half,620);const dusk=ctx.createLinearGradient(0,560,0,1180);dusk.addColorStop(0,'rgba(193,77,79,.55)');dusk.addColorStop(1,'rgba(245,195,102,.24)');ctx.fillStyle=dusk;ctx.fillRect(right,560,half,620);sun(305,760,65,p);sun(780,940,55,p);rc.line(180,1120,470,740,{stroke:C.blue,strokeWidth:4,roughness:1.1,seed:231});rc.line(640,1120,900,900,{stroke:C.red,strokeWidth:4,roughness:1.1,seed:232});text('白天',305,1230,{size:26,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});text('日落',780,1230,{size:26,weight:700,family:'hand',color:C.red,align:'center',alpha:p});text('短路径',305,1275,{size:21,weight:500,family:'mono',color:C.blue,align:'center',alpha:p});text('长路径',780,1275,{size:21,weight:500,family:'mono',color:C.red,align:'center',alpha:p});}
function drawTakeaway(scene,t,p){sketchPanel(140,560,800,620,250);sun(790,700,85,p*.8);const lines=wrap(scene.title,720,48,700,'hand');lines.forEach((line,i)=>text(line,W/2,780+i*66,{size:48,weight:700,family:'hand',color:C.ink,align:'center',alpha:p}));text(scene.caption,W/2,1040,{size:28,weight:400,color:C.soft,align:'center',alpha:p});rc.line(250,1110+Math.sin(t*1.8)*2,830,1110+Math.sin(t*1.8)*2,{stroke:C.blue,strokeWidth:3,roughness:1.1,seed:260});text('散射：光遇到空气分子后改变方向',W/2,1190,{size:23,weight:600,family:'mono',color:C.blue,align:'center',alpha:p});}
function renderAt(t){const time=clamp(t,0,DURATION);NOW=time;boil=Math.floor(time*8);const scene=sceneAt(time);const p=sceneProgress(scene,time);ctx.clearRect(0,0,W,H);paper(time);frame();header(scene,time);titleBlock(scene,time);switch(scene.type){case 'spectrum':drawSpectrum(scene,time,p);break;case 'scatter':drawScatter(scene,time,p);break;case 'observer':drawObserver(scene,time,p);break;case 'sunset':drawSunset(scene,time,p);break;case 'takeaway':drawTakeaway(scene,time,p);break;default:drawTitleScene(scene,time,p);}footer(scene,time);}
window.renderAt=renderAt;window.__DURATION__=DURATION;window.__FPS__=FPS;window.__AUDIT_META__=(t)=>{const s=sceneAt(clamp(t,0,DURATION));return{title:s.title,caption:s.caption};};
loadData().then(()=>document.fonts.ready.then(()=>{renderAt(0);window.__READY__=true;}));
