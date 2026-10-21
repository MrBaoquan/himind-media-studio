import rough from './vendor/rough.esm.js';

const W = 1080, H = 1920, FPS = 30, DURATION = 72;
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
const IMAGES = {};

const DEFAULT = {
  title: '合肥，有一轮“人造太阳”',
  subtitle: 'EAST 托卡马克如何把太阳的聚变搬到地球',
  source: '来源：中国科学院合肥物质科学研究院 / EAST · ITER',
  scenes: [
    {id:'hook',type:'image',start:0,end:5,kicker:'HEFEI / FUSION',title:'合肥，有一轮“人造太阳”',caption:'它不挂在天空，而是在一座实验装置里。',image:'/assets/hook.jpg',ai:true},
    {id:'east',type:'tokamak',start:5,end:12,kicker:'01 / EAST',title:'它不是太阳，是托卡马克',caption:'EAST：全超导托卡马克核聚变实验装置。',labels:['真空室','等离子体','超导磁体']},
    {id:'fusion',type:'fusion',start:12,end:21,kicker:'02 / FUSION',title:'太阳的能量，来自聚变',caption:'轻原子核在极高温下结合，释放能量。',labels:['氢核','氦','中子','能量']},
    {id:'plasma',type:'image',start:21,end:29,kicker:'03 / 100 MILLION °C',title:'上亿度，物质变成等离子体',caption:'没有材料能直接盛放它。',image:'/assets/plasma.jpg',ai:true},
    {id:'magnetic',type:'magnetic',start:29,end:40,kicker:'04 / MAGNETIC CONFINEMENT',title:'没有容器，只有磁场',caption:'强磁场把带电粒子悬在空中，不接触器壁。',labels:['磁力线','带电粒子','真空室']},
    {id:'superconducting',type:'tokamak',start:40,end:48,kicker:'05 / SUPERCONDUCTING',title:'超导磁体，让强磁场长时间工作',caption:'超导线圈让长脉冲运行成为可能。',labels:['超导磁体','等离子体环','真空室']},
    {id:'record',type:'record',start:48,end:58,kicker:'06 / RECORD',title:'1 亿度，1066 秒',caption:'2025-01-20，EAST 刷新此前 403 秒的世界纪录。',labels:['1 亿°C','1066 s','403 s']},
    {id:'boundary',type:'compare',start:58,end:66,kicker:'07 / NOT YET',title:'这不是聚变发电',caption:'它证明的是：我们能让微型太阳稳定燃烧得更久。',labels:['稳定性','材料','氚增殖','净能量增益']},
    {id:'takeaway',type:'image',start:66,end:72,kicker:'TAKEAWAY',title:'照亮下一种可能',caption:'合肥的这轮人造太阳，照亮的不是天空，而是人类能源的下一种可能。',image:'/assets/future.jpg',ai:true,source:'来源：中国科学院合肥物质科学研究院 / EAST · ITER'},
  ],
};

let DATA = DEFAULT;
const C = {paper:'#f7f4ea', paper2:'#fffdf7', ink:'#26313b', soft:'#65727a', faint:'rgba(38,49,59,.12)', blue:'#2b74b5', cyan:'#59b9d1', orange:'#d47c26', red:'#c54a3d', green:'#2d8a68', yellow:'#e6bd3e', violet:'#7652a7'};

function normalizeData(value) {
  if (!value || typeof value !== 'object') return DEFAULT;
  const scenes = Array.isArray(value.scenes) ? value.scenes.filter((s) => s && typeof s === 'object').map((s, i) => ({
    id:String(s.id || `scene-${i+1}`), type:String(s.type || 'title'), start:Number(s.start ?? i*8), end:Number(s.end ?? (i+1)*8),
    kicker:String(s.kicker || ''), title:String(s.title || ''), caption:String(s.caption || ''), note:String(s.note || ''),
    labels:Array.isArray(s.labels) ? s.labels.map(String) : [], image:String(s.image || ''), ai:Boolean(s.ai), source:String(s.source || ''),
  })) : DEFAULT.scenes;
  const subtitles = Array.isArray(value.subtitles) ? value.subtitles.filter((cue) => cue && typeof cue === "object").map((cue) => ({start:Number(cue.start || 0), end:Number(cue.end || 0), text:String(cue.text || ""), beat_id:String(cue.beat_id || "")})).filter((cue) => cue.text && cue.end > cue.start) : [];
  return {title:String(value.title || DEFAULT.title), subtitle:String(value.subtitle || DEFAULT.subtitle), source:String(value.source || DEFAULT.source), scenes:scenes.length?scenes:DEFAULT.scenes, subtitles};
}
async function loadData(){try{const r=await fetch('/input/data.json',{cache:'no-store'});if(r.ok)DATA=normalizeData(await r.json());}catch{DATA=DEFAULT;}}
async function loadImages(){
  const tasks=[];
  for(const scene of DATA.scenes){
    if(!scene.image) continue;
    tasks.push(new Promise((resolve)=>{
      const img=new Image();
      img.onload=()=>{IMAGES[scene.id]=img;resolve();};
      img.onerror=()=>{resolve();};
      img.src=scene.image;
    }));
  }
  await Promise.all(tasks);
}
const clamp=(v,a,b)=>Math.max(a,Math.min(b,v));
const ease=(v)=>1-Math.pow(1-clamp(v,0,1),3);
const hash=(n)=>{const x=Math.sin(n*127.1+311.7)*43758.5453;return x-Math.floor(x);};
function setFont(size,weight=400,family='body'){
  const stack=family==='hand'?'"KaiTi","STKaiti","Segoe Print","Comic Sans MS","Noto Sans SC",sans-serif':family==='mono'?'"Cascadia Mono","JetBrains Mono",Consolas,monospace':'"Microsoft YaHei","Noto Sans SC","PingFang SC",sans-serif';
  ctx.font=`${weight} ${size}px ${stack}`;
}
function text(value,x,y,o={}){const{size=28,weight=400,family='body',color=C.ink,align='left',alpha=1,baseline='alphabetic'}=o;if(alpha<=0)return;ctx.save();ctx.globalAlpha*=alpha;ctx.fillStyle=color;ctx.textAlign=align;ctx.textBaseline=baseline;setFont(size,weight,family);ctx.fillText(String(value),x,y);ctx.restore();}
function wrap(value,maxWidth,size,weight=400,family='body'){setFont(size,weight,family);const chars=[...String(value||'')];const lines=[];let line='';for(const ch of chars){const test=line+ch;if(line&&ctx.measureText(test).width>maxWidth){lines.push(line);line=ch;}else line=test;}if(line)lines.push(line);return lines;}
function sceneAt(t){for(const s of DATA.scenes){if(t>=s.start&&t<s.end)return s;}return DATA.scenes[DATA.scenes.length-1];}
function sceneProgress(scene,t){return clamp((t-scene.start)/Math.max(.001,scene.end-scene.start),0,1);}
function paper(t){ctx.fillStyle=C.paper;ctx.fillRect(0,0,W,H);const gx=W*.72+Math.sin(t*.33)*14,gy=H*.18+Math.cos(t*.27)*10;const g=ctx.createRadialGradient(gx,gy,20,gx,gy,W*.9);g.addColorStop(0,'rgba(174,207,221,.22)');g.addColorStop(.55,'rgba(221,214,190,.09)');g.addColorStop(1,'rgba(255,255,255,0)');ctx.fillStyle=g;ctx.fillRect(0,0,W,H);const dx=Math.sin(t*.42)*3,dy=Math.cos(t*.35)*2;ctx.save();ctx.strokeStyle='rgba(38,49,59,.025)';ctx.lineWidth=1;for(let y=80;y<H;y+=42){ctx.beginPath();ctx.moveTo(70,y+dy*.4);ctx.lineTo(W-70,y+dy*.4);ctx.stroke();}for(let i=0;i<240;i++){const x=hash(i+1)*W+dx,y=hash(i+101)*H+dy;ctx.fillStyle=`rgba(38,49,59,${(.012+hash(i+201)*.018).toFixed(3)})`;ctx.fillRect(x,y,2,2);}ctx.restore();}
function frame(){rc.rectangle(48,48,W-96,H-96,{stroke:'rgba(38,49,59,.46)',strokeWidth:3,roughness:1.35,bowing:1.15,seed:11});}
function header(scene,t){const p=ease((t-scene.start)/.65);const light=scene.type==='image';const ink=light?'#ffffff':C.ink;const accent=light?'#9dd7ff':C.blue;text('HIMIND / SCIENCE FUSION',92,112,{size:21,weight:700,family:'mono',color:ink,alpha:p});text(scene.kicker||'SCIENCE FUSION',W-92,112,{size:20,weight:700,family:'mono',color:accent,align:'right',alpha:p});rc.line(92,132,W-92,132,{stroke:light?'rgba(255,255,255,.35)':'rgba(38,49,59,.25)',strokeWidth:2,roughness:1.1,seed:12});}
function titleBlock(scene,t){const p=ease((t-scene.start)/.7);if(p<=0)return;const size=[...String(scene.title)].length>14?44:54;text(scene.title,W/2,265,{size,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});const lines=wrap(scene.caption,W-200,27,400,'body');lines.forEach((line,i)=>text(line,W/2,325+i*38,{size:27,weight:400,color:C.soft,align:'center',alpha:p}));}
function footer(scene,t){const src=scene.source||DATA.source;const color=scene.type==='image'?'rgba(255,255,255,.72)':C.soft;if(src)text(src,W/2,1778,{size:20,weight:400,color,align:'center',alpha:ease((t-scene.start)/.8)});text('SCIENCE / FUSION / HEFEI',92,1814,{size:19,weight:600,family:'mono',color,alpha:.8});}
function drawSubtitles(t){
  const cues=DATA.subtitles||[];const cue=cues.find((item)=>t>=item.start&&t<=item.end);if(!cue)return;
  const scene=sceneAt(t);const light=scene.type==='image';const fade=Math.min(1,(t-cue.start)/.16,(cue.end-t)/.18);if(fade<=0)return;
  const size=38;const lines=wrap(cue.text,880,size,600,'body');const lineHeight=54;const cardH=lines.length*lineHeight+28;const cardW=940;const x=(W-cardW)/2;const y=1690-cardH;const radius=18;
  ctx.save();ctx.globalAlpha=fade;
  ctx.fillStyle=light?'rgba(7,11,16,.74)':'rgba(255,253,247,.90)';ctx.strokeStyle=light?'rgba(157,215,255,.34)':'rgba(38,49,59,.20)';ctx.lineWidth=2;
  ctx.beginPath();ctx.roundRect(x,y,cardW,cardH,radius);ctx.fill();ctx.stroke();
  lines.forEach((line,index)=>text(line,W/2,y+52+index*lineHeight,{size,weight:600,family:'body',color:light?'#ffffff':C.ink,align:'center',alpha:fade}));
  ctx.restore();
}
function sun(cx,cy,r,alpha=1,color=C.orange){ctx.save();ctx.globalAlpha*=alpha;const R=r*(1+Math.sin(NOW*1.9)*.018);const g=ctx.createRadialGradient(cx,cy,R*.2,cx,cy,R);g.addColorStop(0,'#ffe58a');g.addColorStop(.55,color);g.addColorStop(1,'rgba(244,186,77,.08)');ctx.fillStyle=g;ctx.beginPath();ctx.arc(cx,cy,R,0,Math.PI*2);ctx.fill();rc.circle(cx,cy,R*.72,{stroke:color,strokeWidth:3,roughness:1.3,seed:41});ctx.restore();}
function arrow(x1,y1,x2,y2,color=C.blue,seed=1,width=3){rc.line(x1,y1,x2,y2,{stroke:color,strokeWidth:width,roughness:1.2,seed});const a=Math.atan2(y2-y1,x2-x1);rc.line(x2,y2,x2-Math.cos(a-.45)*16,y2-Math.sin(a-.45)*16,{stroke:color,strokeWidth:width,roughness:1,seed:seed+1});rc.line(x2,y2,x2-Math.cos(a+.45)*16,y2-Math.sin(a+.45)*16,{stroke:color,strokeWidth:width,roughness:1,seed:seed+2});}
function sketchPanel(x,y,w,h,seed=1,color='rgba(43,116,181,.18)'){ctx.save();ctx.fillStyle='rgba(255,253,247,.72)';ctx.fillRect(x,y,w,h);rc.rectangle(x,y,w,h,{stroke:color,strokeWidth:2.4,roughness:1.45,bowing:1.2,seed});ctx.restore();}
function label(textValue,x,y,color=C.ink,align='center',size=24){text(textValue,x,y,{size,weight:600,family:'hand',color,align});}

function drawImageScene(scene,t,p){
  const img=IMAGES[scene.id];
  if(img){
    const zoom=1.02+0.06*ease(p);
    const scale=Math.max(W/img.width,H/img.height)*zoom;
    const dw=img.width*scale, dh=img.height*scale;
    const dx=(W-dw)/2+Math.sin(t*.2)*6;
    const dy=(H-dh)/2-12*ease(p);
    ctx.drawImage(img,dx,dy,dw,dh);
    const grad=ctx.createLinearGradient(0,0,0,H);
    grad.addColorStop(0,'rgba(8,14,22,.42)');grad.addColorStop(.45,'rgba(8,14,22,.04)');grad.addColorStop(1,'rgba(8,14,22,.66)');
    ctx.fillStyle=grad;ctx.fillRect(0,0,W,H);
  }else{
    const g=ctx.createLinearGradient(0,0,W,H);g.addColorStop(0,'#0b1220');g.addColorStop(1,'#1d4ed8');ctx.fillStyle=g;ctx.fillRect(0,0,W,H);
    text('IMAGE PLACEHOLDER',W/2,H/2,{size:34,weight:700,family:'mono',color:'rgba(255,255,255,.5)',align:'center'});
  }
  if(scene.ai) text('AI 生成示意图',W-92,180,{size:18,weight:600,family:'mono',color:'rgba(255,255,255,.78)',align:'right'});
  const size=[...String(scene.title)].length>14?44:54;
  text(scene.title,W/2,285,{size,weight:700,family:'hand',color:'#ffffff',align:'center',alpha:ease(p/.6)});
  const cardY=1490;
  ctx.save();ctx.fillStyle='rgba(8,14,22,.66)';ctx.fillRect(92,cardY,W-184,220);ctx.strokeStyle='rgba(255,255,255,.18)';ctx.lineWidth=2;ctx.strokeRect(92,cardY,W-184,220);ctx.restore();
  const lines=wrap(scene.caption,W-220,28,500,'body');
  lines.forEach((line,i)=>text(line,W/2,cardY+78+i*40,{size:28,weight:500,color:'#ffffff',align:'center',alpha:ease(p/.7)}));
}

function drawTokamak(scene,t,p){
  const cx=540,cy=980;
  const glow=ctx.createRadialGradient(cx,cy,40,cx,cy,390);
  glow.addColorStop(0,'rgba(212,124,38,.30)');glow.addColorStop(.55,'rgba(43,116,181,.14)');glow.addColorStop(1,'rgba(255,255,255,0)');
  ctx.fillStyle=glow;ctx.beginPath();ctx.ellipse(cx,cy,380,270,0,0,Math.PI*2);ctx.fill();
  rc.ellipse(cx,cy,660,440,{stroke:C.ink,strokeWidth:4,roughness:1.2,seed:210});
  rc.ellipse(cx,cy,460,290,{stroke:C.blue,strokeWidth:3,roughness:1.2,seed:211});
  ctx.save();ctx.globalAlpha=p*.92;ctx.strokeStyle=C.orange;ctx.lineWidth=16;ctx.beginPath();ctx.ellipse(cx,cy,270,175,0,0,Math.PI*2);ctx.stroke();ctx.strokeStyle='rgba(255,255,255,.55)';ctx.lineWidth=4;ctx.stroke();ctx.restore();
  for(let i=0;i<8;i++){
    const a=i*Math.PI/4;
    const x=cx+Math.cos(a)*390, y=cy+Math.sin(a)*260;
    rc.rectangle(x-24,y-62,48,124,{stroke:C.violet,strokeWidth:4,roughness:1.2,seed:220+i});
  }
  arrow(cx-300,cy-200,210,640,C.blue,240,2.6);label('真空室',170,610,C.ink,'center',24);
  arrow(cx+260,cy-160,820,690,C.orange,250,2.6);label('等离子体',820,660,C.orange,'center',24);
  arrow(cx+350,cy+120,850,1280,C.violet,260,2.6);label('超导磁体',850,1330,C.violet,'center',24);
  text('磁场把上亿度的等离子体悬在空中',W/2,1500,{size:26,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
}

function drawFusion(scene,t,p){
  const cx=540,cy=940;
  const pulse=1+Math.sin(NOW*2.2)*.03;
  const g=ctx.createRadialGradient(cx,cy,20,cx,cy,180*pulse);
  g.addColorStop(0,'rgba(230,189,62,.85)');g.addColorStop(.5,'rgba(212,124,38,.30)');g.addColorStop(1,'rgba(255,255,255,0)');
  ctx.fillStyle=g;ctx.beginPath();ctx.arc(cx,cy,180*pulse,0,Math.PI*2);ctx.fill();
  rc.circle(cx-190,cy,72,{stroke:C.blue,strokeWidth:4,roughness:1.2,seed:301});
  rc.circle(cx+190,cy,72,{stroke:C.blue,strokeWidth:4,roughness:1.2,seed:302});
  text('+',cx-190,cy+22,{size:64,weight:700,family:'hand',color:C.blue,align:'center'});
  text('+',cx+190,cy+22,{size:64,weight:700,family:'hand',color:C.blue,align:'center'});
  arrow(cx-110,cy,cx-40,cy,C.red,310,2.6);arrow(cx+110,cy,cx+40,cy,C.red,311,2.6);
  text('聚变',cx,cy+18,{size:48,weight:700,family:'hand',color:C.orange,align:'center',alpha:p});
  for(let i=0;i<12;i++){const a=i*Math.PI/6;rc.line(cx+Math.cos(a)*120,cy+Math.sin(a)*120,cx+Math.cos(a)*190,cy+Math.sin(a)*190,{stroke:C.yellow,strokeWidth:4,roughness:1.1,seed:320+i});}
  rc.circle(cx-110,cy+300,56,{stroke:C.green,strokeWidth:4,roughness:1.2,seed:340});
  text('氦',cx-110,cy+318,{size:34,weight:700,family:'hand',color:C.green,align:'center'});
  rc.circle(cx+140,cy+300,34,{stroke:C.soft,strokeWidth:4,roughness:1.2,seed:341});
  text('中子',cx+140,cy+365,{size:26,weight:600,family:'hand',color:C.soft,align:'center'});
  text('能量',cx+130,cy+200,{size:26,weight:700,family:'hand',color:C.orange,align:'center',alpha:p});
  label('氢核',cx-190,cy-130,C.blue,'center',24);label('氢核',cx+190,cy-130,C.blue,'center',24);
  text('轻核结合，质量减少，能量释放',W/2,1490,{size:26,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
}

function drawMagnetic(scene,t,p){
  const cx=540,cy=980;
  for(let k=0;k<5;k++){
    rc.ellipse(cx,cy,520+k*56,320+k*38,{stroke:k%2?C.cyan:C.blue,strokeWidth:2.4,roughness:1.2,seed:400+k});
  }
  ctx.save();ctx.strokeStyle=C.orange;ctx.lineWidth=3;ctx.beginPath();
  for(let i=0;i<220;i++){
    const a=i/220*Math.PI*8 + NOW*.9;
    const r=30+i*.95;
    const x=cx+Math.cos(a)*r*1.15, y=cy+Math.sin(a)*r*.72;
    if(i===0)ctx.moveTo(x,y);else ctx.lineTo(x,y);
  }
  ctx.stroke();ctx.restore();
  const a=220/220*Math.PI*8 + NOW*.9;const r=30+219*.95;
  const px=cx+Math.cos(a)*r*1.15,py=cy+Math.sin(a)*r*.72;
  ctx.save();ctx.fillStyle=C.orange;ctx.beginPath();ctx.arc(px,py,10,0,Math.PI*2);ctx.fill();ctx.restore();
  arrow(180,680,300,780,C.blue,420,2.6);label('磁力线',170,640,C.blue,'center',24);
  arrow(860,760,700,860,C.orange,421,2.6);label('带电粒子',860,720,C.orange,'center',24);
  label('真空室',540,1320,C.ink,'center',26);
  text('粒子沿磁力线螺旋运动，不接触器壁',W/2,1490,{size:26,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
}

function drawRecord(scene,t,p){
  const n=Math.round(1066*ease(p));
  text('1 亿°C',W/2,620,{size:40,weight:700,family:'hand',color:C.orange,align:'center',alpha:p});
  text(String(n),W/2+20,930,{size:196,weight:700,family:'hand',color:C.ink,align:'right',alpha:p});
  text('s',W/2+45,930,{size:88,weight:700,family:'hand',color:C.blue,align:'left',alpha:p});
  text('稳态长脉冲高约束模运行',W/2,1020,{size:28,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
  const x0=160,x1=920,y=1240;
  rc.line(x0,y,x1,y,{stroke:C.ink,strokeWidth:4,roughness:1.1,seed:500});
  const oldX=x0+(x1-x0)*(403/1066), newX=x1;
  rc.line(oldX,y-34,oldX,y+34,{stroke:C.soft,strokeWidth:4,roughness:1.1,seed:501});
  rc.line(newX,y-46,newX,y+46,{stroke:C.orange,strokeWidth:6,roughness:1.1,seed:502});
  text('403 s',oldX,y+86,{size:28,weight:600,family:'mono',color:C.soft,align:'center',alpha:p});
  text('1066 s',newX,y+86,{size:28,weight:700,family:'mono',color:C.orange,align:'center',alpha:p});
  text('此前纪录',oldX,y-60,{size:22,weight:500,family:'hand',color:C.soft,align:'center',alpha:p});
  text('2025-01-20',newX,y-70,{size:22,weight:600,family:'mono',color:C.orange,align:'center',alpha:p});
  text('不是“已经发电”，而是“稳定控制得更久”',W/2,1490,{size:26,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
}

function drawCompare(scene,t,p){
  sketchPanel(100,600,410,430,600,'rgba(43,116,181,.22)');
  sketchPanel(570,600,410,430,610,'rgba(101,114,122,.22)');
  label('实验室纪录',305,690,C.blue,'center',30);
  text('1066 s',305,850,{size:64,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});
  text('1 亿°C',305,940,{size:32,weight:600,family:'hand',color:C.orange,align:'center',alpha:p});
  label('商业发电',775,690,C.soft,'center',30);
  text('连续运行',775,850,{size:52,weight:700,family:'hand',color:C.ink,align:'center',alpha:p});
  text('净能量增益',775,940,{size:30,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
  const labels=scene.labels.length?scene.labels:['稳定性','材料','氚增殖','净能量增益'];
  const colors=[C.blue,C.green,C.violet,C.orange];
  labels.forEach((item,i)=>{const x=150+(i%2)*390,y=1220+Math.floor(i/2)*140;sketchPanel(x,y,330,100,620+i,colors[i%colors.length]);text(item,x+165,y+62,{size:26,weight:600,family:'hand',color:colors[i%colors.length],align:'center',alpha:p});});
  text('距离“用聚变点亮城市”，还有工程难题',W/2,1580,{size:26,weight:600,family:'hand',color:C.soft,align:'center',alpha:p});
}


function renderAt(t){
  const time=clamp(t,0,DURATION);
  NOW=time; boil=Math.floor(time*8);
  const scene=sceneAt(time);
  const p=sceneProgress(scene,time);
  ctx.clearRect(0,0,W,H);
  if(scene.type==='image') drawImageScene(scene,time,p); else paper(time);
  frame();
  header(scene,time);
  if(scene.type!=='image') titleBlock(scene,time);
  switch(scene.type){
    case 'image': break;
    case 'tokamak': drawTokamak(scene,time,p); break;
    case 'fusion': drawFusion(scene,time,p); break;
    case 'magnetic': drawMagnetic(scene,time,p); break;
    case 'record': drawRecord(scene,time,p); break;
    case 'compare': drawCompare(scene,time,p); break;
    default: break;
  }
  footer(scene,time);
  drawSubtitles(time);
}

window.renderAt=renderAt;window.__DURATION__=DURATION;window.__FPS__=FPS;window.__AUDIT_META__=(t)=>{const s=sceneAt(clamp(t,0,DURATION));return{title:s.title,caption:s.caption};};
loadData().then(loadImages).then(()=>document.fonts.ready.then(()=>{renderAt(0);window.__READY__=true;}));
