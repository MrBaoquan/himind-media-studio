const W = 1080, H = 1920, FPS = 30, DURATION = 10;
const canvas = document.getElementById('stage');
const ctx = canvas.getContext('2d');
const C = {paper:'#e9ddc7', ink:'#27231f', muted:'#776b5b', red:'#b4472f', blue:'#2f5d7a'};
const clamp=(v,a,b)=>Math.max(a,Math.min(b,v));
const ease=(v)=>1-Math.pow(1-clamp(v,0,1),3);
const hash=(n)=>{const x=Math.sin(n*127.1+311.7)*43758.5453;return x-Math.floor(x);};
function bg(t){
  ctx.fillStyle=C.paper; ctx.fillRect(0,0,W,H);
  for(let i=0;i<700;i++){
    const x=hash(i+1)*W,y=hash(i+101)*H,l=4+hash(i+201)*28;
    ctx.strokeStyle=`rgba(70,55,40,${(0.02+hash(i+301)*0.03).toFixed(3)})`;
    ctx.lineWidth=1;ctx.beginPath();ctx.moveTo(x,y);ctx.lineTo(x+l,y+(hash(i+401)-.5)*3);ctx.stroke();
  }
  const g=ctx.createRadialGradient(W*.7,H*.2,0,W*.7,H*.2,W*.9);
  g.addColorStop(0,'rgba(150,110,65,.09)');g.addColorStop(1,'rgba(150,110,65,0)');
  ctx.fillStyle=g;ctx.fillRect(0,0,W,H);
}
function torn(x,y,w,h,rot,color,seed){
  ctx.save();ctx.translate(x+w/2,y+h/2);ctx.rotate(rot);ctx.translate(-w/2,-h/2);
  ctx.shadowColor='rgba(50,35,20,.25)';ctx.shadowBlur=18;ctx.shadowOffsetY=10;ctx.fillStyle=color;
  ctx.beginPath();
  ctx.moveTo(0+(hash(seed)-.5)*12,0+(hash(seed+1)-.5)*12);
  for(let i=1;i<=8;i++)ctx.lineTo(w*i/8+(hash(seed+i+2)-.5)*12,(hash(seed+i+12)-.5)*10);
  for(let i=1;i<=8;i++)ctx.lineTo(w+(hash(seed+i+22)-.5)*12,h*i/8+(hash(seed+i+32)-.5)*12);
  for(let i=1;i<=8;i++)ctx.lineTo(w-w*i/8+(hash(seed+i+42)-.5)*12,h+(hash(seed+i+52)-.5)*12);
  for(let i=1;i<=8;i++)ctx.lineTo((hash(seed+i+62)-.5)*12,h-h*i/8+(hash(seed+i+72)-.5)*12);
  ctx.closePath();ctx.fill();ctx.shadowBlur=0;ctx.shadowOffsetY=0;
  ctx.strokeStyle='rgba(90,70,50,.14)';ctx.lineWidth=2;ctx.stroke();ctx.restore();
}
function tape(x,y,w,h,rot){
  ctx.save();ctx.translate(x,y);ctx.rotate(rot);ctx.fillStyle='rgba(229,205,145,.72)';ctx.fillRect(-w/2,-h/2,w,h);
  ctx.strokeStyle='rgba(120,95,50,.16)';ctx.lineWidth=1;
  for(let i=-w/2;i<w/2;i+=12){ctx.beginPath();ctx.moveTo(i,-h/2);ctx.lineTo(i+5,h/2);ctx.stroke();}
  ctx.restore();
}
function chrome(){
  ctx.fillStyle=C.ink;ctx.font='600 24px "Courier New",monospace';ctx.fillText('HIMIND / PAPER COLLAGE',76,124);
  ctx.textAlign='right';ctx.fillStyle=C.red;ctx.fillText('ARCHIVE 2026',W-76,124);ctx.textAlign='left';
  ctx.strokeStyle='rgba(39,35,31,.35)';ctx.lineWidth=2;ctx.beginPath();ctx.moveTo(76,148);ctx.lineTo(330,148);ctx.stroke();
}
function title(t){
  const p=ease(t/1.6);if(p<=0)return;ctx.save();ctx.globalAlpha=p;ctx.textAlign='center';
  ctx.fillStyle=C.red;ctx.font='600 24px "Courier New",monospace';ctx.letterSpacing='8px';ctx.fillText('A PAPER STORY',W/2,520+20*(1-p));
  ctx.fillStyle=C.ink;ctx.font='700 118px Georgia,serif';ctx.fillText('城市记忆',W/2+3,690+40*(1-p));
  ctx.strokeStyle=C.red;ctx.lineWidth=6;ctx.beginPath();ctx.moveTo(W/2-210*p,735);ctx.lineTo(W/2+210*p,735);ctx.stroke();
  ctx.fillStyle=C.muted;ctx.font='500 28px "Courier New",monospace';ctx.fillText('FRAGMENTS / PLACES / PEOPLE',W/2,790);ctx.restore();
}
function card(i,t,x,y,w,h,rot,color,name,cap){
  const p=ease((t-(2+i*.7))/.8);if(p<=0)return;ctx.save();ctx.globalAlpha=p;ctx.translate(0,60*(1-p));
  torn(x,y,w,h,rot,color,100+i*17);
  ctx.save();ctx.translate(x+w/2,y+h/2);ctx.rotate(rot);
  ctx.fillStyle=['#7f9181','#7e91a8','#a9806e'][i];ctx.fillRect(-w/2+26,-h/2+26,w-52,h-170);
  ctx.fillStyle='rgba(255,244,220,.45)';ctx.beginPath();ctx.arc(-w/6,-h/6,54+8*Math.sin(t+i),0,Math.PI*2);ctx.fill();
  ctx.strokeStyle='rgba(255,244,220,.55)';ctx.lineWidth=4;
  for(let j=0;j<3;j++){ctx.beginPath();ctx.moveTo(-w/2+38,-h/2+110+j*42);ctx.lineTo(w/2-38,-h/2+90+j*42);ctx.stroke();}
  ctx.fillStyle=C.ink;ctx.textAlign='left';ctx.font='700 42px Georgia,serif';ctx.fillText(name,-w/2+30,h/2-76);
  ctx.fillStyle=C.muted;ctx.font='500 21px "Courier New",monospace';ctx.fillText(cap,-w/2+30,h/2-38);ctx.restore();
  tape(x+w/2,y+10,150,38,rot-.2);ctx.restore();
}
function cards(t){card(0,t,110,960,390,500,-.08,'#efe3cf','01 / 街角','CORNER / 2008');card(1,t,360,1080,390,500,.06,'#e6d3bd','02 / 站台','PLATFORM / 2014');card(2,t,610,900,390,500,-.03,'#ead8c8','03 / 人群','CROWD / 2026');}
function stamp(t){
  const p=ease((t-6.2)/.6);if(p<=0)return;ctx.save();ctx.globalAlpha=p;ctx.translate(820,1580);ctx.rotate(-.18);
  ctx.strokeStyle=C.red;ctx.lineWidth=6;ctx.beginPath();ctx.arc(0,0,105,0,Math.PI*2);ctx.stroke();
  ctx.beginPath();ctx.arc(0,0,82,0,Math.PI*2);ctx.stroke();ctx.fillStyle=C.red;ctx.textAlign='center';
  ctx.font='700 26px "Courier New",monospace';ctx.fillText('ARCHIVE',0,-36);ctx.font='700 42px Georgia,serif';ctx.fillText('2026',0,30);ctx.restore();
}
function lines(t){
  const p=ease((t-6)/1);if(p<=0)return;ctx.save();ctx.globalAlpha=p;ctx.strokeStyle=C.blue;ctx.lineWidth=4;
  ctx.beginPath();ctx.moveTo(120,1640);ctx.bezierCurveTo(260,1600,300,1700,440,1640);ctx.stroke();
  ctx.fillStyle=C.blue;ctx.font='italic 30px Georgia,serif';ctx.fillText('places stay in fragments.',130,1715);ctx.restore();
}
function finale(t){
  const p=ease((t-8.2)/.7);if(p<=0)return;ctx.save();ctx.globalAlpha=p;ctx.fillStyle=C.ink;ctx.textAlign='center';
  ctx.font='700 46px Georgia,serif';ctx.fillText('把记忆，剪成画面。',W/2,1800);ctx.restore();
}
function renderAt(t){const time=clamp(t,0,DURATION);ctx.clearRect(0,0,W,H);bg(time);chrome();title(time);cards(time);lines(time);stamp(time);finale(time);}
window.renderAt=renderAt;window.__DURATION__=DURATION;window.__FPS__=FPS;
document.fonts.ready.then(()=>{renderAt(0);window.__READY__=true;});
