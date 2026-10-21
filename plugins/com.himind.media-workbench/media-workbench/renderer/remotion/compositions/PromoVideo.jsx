import React from 'react';
import {AbsoluteFill, Img, Sequence, interpolate, spring, useCurrentFrame, useVideoConfig} from 'remotion';
import {PROMO_ASSETS} from '../promo-assets.js';

const C = {
  bg: '#071018',
  bg2: '#0b1722',
  panel: 'rgba(255,255,255,.055)',
  line: 'rgba(255,255,255,.11)',
  text: '#f5f7fa',
  muted: '#9aa7b4',
  accent: '#4fd1c5',
  blue: '#5aa9ff',
  warm: '#ffb86b',
};

const FONT = '"Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", sans-serif';

const useReveal = (delay = 0, damping = 18) => {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  return spring({frame: frame - delay, fps, config: {damping, stiffness: 90, mass: .7}});
};

const GRAIN_URI = `data:image/svg+xml;utf8,${encodeURIComponent("<svg xmlns='http://www.w3.org/2000/svg' width='160' height='160'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='.86' numOctaves='3' stitchTiles='stitch'/><feColorMatrix type='saturate' values='0'/></filter><rect width='100%' height='100%' filter='url(#n)' opacity='.55'/></svg>")}`;

const Grain = () => (
  <AbsoluteFill style={{
    pointerEvents: 'none',
    backgroundImage: `url("${GRAIN_URI}"), repeating-linear-gradient(135deg, rgba(255,255,255,.10) 0 1px, rgba(0,0,0,0) 1px 4px)`,
    backgroundSize: '160px 160px, auto',
    backgroundRepeat: 'repeat, repeat',
    opacity: .42,
    mixBlendMode: 'screen',
  }} />
);
const Grid = () => {
  const frame = useCurrentFrame();
  const offset = (frame * .35) % 64;
  return <AbsoluteFill style={{opacity: .5, backgroundImage: `linear-gradient(rgba(154,167,180,.07) 1px, transparent 1px), linear-gradient(90deg, rgba(154,167,180,.07) 1px, transparent 1px)`, backgroundSize: '64px 64px', backgroundPosition: `${offset}px ${offset}px`}} />;
};

const Glow = ({color = C.accent, x = '72%', y = '28%', size = 900}) => (
  <div style={{position: 'absolute', left: x, top: y, width: size, height: size, transform: 'translate(-50%, -50%)', borderRadius: '50%', background: `radial-gradient(circle, ${color}28 0%, transparent 68%)`, filter: 'blur(8px)'}} />
);

const Kicker = ({children}) => <div style={{fontSize: 15, letterSpacing: '.28em', textTransform: 'uppercase', color: C.accent, fontWeight: 700}}>{children}</div>;

const HeroScene = ({scene}) => {
  const reveal = useReveal(2);
  const sub = useReveal(12);
  return <AbsoluteFill style={{background: `radial-gradient(1200px 700px at 72% 25%, #123244 0%, ${C.bg} 62%)`, color: C.text, fontFamily: FONT, overflow: 'hidden'}}>
    <Grid />
    <Glow />
    <Glow color={C.blue} x="15%" y="80%" size={700} />
    <div style={{position: 'absolute', left: 150, top: 180, right: 150}}>
      <Kicker>{scene.kicker || 'HIMIND AGENT · AUTOMEDIA WORKBENCH'}</Kicker>
      <div style={{marginTop: 34, fontSize: 150, lineHeight: .95, fontWeight: 850, letterSpacing: '-.055em', transform: `translateY(${interpolate(reveal, [0,1], [28,0])}px)`, opacity: reveal}}>{scene.title || 'HiMind'}</div>
      <div style={{marginTop: 38, maxWidth: 1060, fontSize: 42, lineHeight: 1.35, color: C.muted, opacity: sub, transform: `translateY(${interpolate(sub, [0,1], [20,0])}px)`}}>{scene.subtitle || '让 AI 成为你的创作团队'}</div>
    </div>
    <div style={{position: 'absolute', left: 154, bottom: 220, display: 'flex', gap: 28, color: C.muted, fontSize: 18}}>
      <span>CreativePackage</span><span style={{color: C.accent}}>→</span><span>Plan</span><span style={{color: C.accent}}>→</span><span>Timeline</span><span style={{color: C.accent}}>→</span><span>Render</span>
    </div>
  </AbsoluteFill>;
};

const StatementCard = ({label, index}) => {
  const r = useReveal(14 + index * 8);
  return <div style={{flex: 1, minHeight: 210, padding: 34, border: `1px solid ${C.line}`, borderRadius: 20, background: C.panel, transform: `translateY(${interpolate(r, [0,1], [34,0])}px)`, opacity: r}}>
    <div style={{fontSize: 15, color: index === 0 ? C.accent : index === 1 ? C.blue : C.warm, letterSpacing: '.18em', fontWeight: 700}}>0{index + 1}</div>
    <div style={{marginTop: 26, fontSize: 38, fontWeight: 760}}>{label}</div>
    <div style={{marginTop: 14, fontSize: 19, color: C.muted, lineHeight: 1.5}}>{index === 0 ? '把想法写成可审阅的创作脉络' : index === 1 ? '把脉络拆成可执行的镜头与节奏' : '把镜头放进可编辑的时间线'}</div>
  </div>;
};

const StatementScene = ({scene}) => {
  const reveal = useReveal(2);
  const cards = scene.points || ['Brief', 'Storyboard', 'Timeline'];
  return <AbsoluteFill style={{background: `linear-gradient(135deg, ${C.bg} 0%, ${C.bg2} 100%)`, color: C.text, fontFamily: FONT, overflow: 'hidden'}}>
    <Grid />
    <Glow color={C.blue} x="85%" y="18%" size={800} />
    <div style={{position: 'absolute', left: 150, top: 150, right: 150}}>
      <Kicker>{scene.kicker || 'ONE REQUEST · ONE CREATIVE THREAD'}</Kicker>
      <div style={{marginTop: 30, fontSize: 74, lineHeight: 1.16, fontWeight: 800, letterSpacing: '-.035em', maxWidth: 1500, opacity: reveal}}>{scene.title || '从一句话，到一条完整的片子'}</div>
      <div style={{marginTop: 26, fontSize: 28, color: C.muted, opacity: reveal}}>{scene.subtitle || '调研 · 脚本 · 分镜 · 素材 · 剪辑 · 渲染'}</div>
    </div>
    <div style={{position: 'absolute', left: 150, right: 150, top: 620, display: 'flex', gap: 28}}>
      {cards.map((label, index) => <StatementCard key={label} label={label} index={index} />)}
    </div>
  </AbsoluteFill>;
};

const ScreenshotScene = ({scene, durationFrames}) => {
  const frame = useCurrentFrame();
  const {durationInFrames: compositionDuration} = useVideoConfig();
  const durationInFrames = Math.max(1, Number(durationFrames) || compositionDuration);
  const reveal = useReveal(2);
  const zoom = interpolate(frame, [0, durationInFrames], [1.06, 1.13], {extrapolateRight: 'clamp'});
  const image = scene.image || 'timeline.png';
  const imageSrc = String(image).startsWith('data:') ? image : (PROMO_ASSETS[image] || PROMO_ASSETS['timeline.png']);
  const bullets = scene.bullets || [];
  return <AbsoluteFill style={{background: `linear-gradient(125deg, ${C.bg} 0%, #0a1822 58%, #071018 100%)`, color: C.text, fontFamily: FONT, overflow: 'hidden'}}>
    <Grid />
    <Glow color={C.blue} x="78%" y="30%" size={900} />
    <div style={{position: 'absolute', left: 108, top: 110, width: 620, opacity: reveal, transform: `translateX(${interpolate(reveal, [0,1], [-24,0])}px)`}}>
      <Kicker>{scene.kicker || 'REAL WORKBENCH · REAL OUTPUT'}</Kicker>
      <div style={{marginTop: 28, fontSize: 64, lineHeight: 1.14, fontWeight: 820, letterSpacing: '-.035em'}}>{scene.title}</div>
      <div style={{marginTop: 22, fontSize: 25, lineHeight: 1.5, color: C.muted}}>{scene.subtitle}</div>
      <div style={{marginTop: 42, display: 'flex', flexDirection: 'column', gap: 18}}>
        {bullets.map((bullet, index) => <div key={bullet} style={{display: 'flex', alignItems: 'center', gap: 14, fontSize: 20, color: index === 0 ? C.text : C.muted}}><span style={{width: 9, height: 9, borderRadius: 99, background: C.accent, boxShadow: `0 0 18px ${C.accent}`}} />{bullet}</div>)}
      </div>
    </div>
    <div style={{position: 'absolute', left: 760, top: 130, width: 1060, height: 642, borderRadius: 22, overflow: 'hidden', border: `1px solid ${C.line}`, boxShadow: '0 38px 90px rgba(0,0,0,.52)', background: '#0b1118', transform: `scale(${zoom})`, transformOrigin: 'center center'}}>
      <div style={{height: 46, display: 'flex', alignItems: 'center', gap: 9, padding: '0 18px', borderBottom: `1px solid ${C.line}`, background: '#0e1720'}}>
        <span style={{width: 10, height: 10, borderRadius: 99, background: '#ff6b6b'}} /><span style={{width: 10, height: 10, borderRadius: 99, background: '#ffd166'}} /><span style={{width: 10, height: 10, borderRadius: 99, background: '#4fd1c5'}} />
        <span style={{marginLeft: 16, color: C.muted, fontSize: 14, letterSpacing: '.08em'}}>HiMind Agent / 自媒体创作台</span>
      </div>
      <Img src={imageSrc} style={{width: '100%', height: 'calc(100% - 46px)', objectFit: 'cover', objectPosition: 'top center'}} />
    </div>
  </AbsoluteFill>;
};

const CTAScene = ({scene}) => {
  const reveal = useReveal(2);
  const sub = useReveal(12);
  return <AbsoluteFill style={{background: `radial-gradient(1000px 620px at 50% 35%, #123244 0%, ${C.bg} 68%)`, color: C.text, fontFamily: FONT, alignItems: 'center', justifyContent: 'center', textAlign: 'center', overflow: 'hidden'}}>
    <Grid />
    <Glow color={C.accent} x="50%" y="38%" size={980} />
    <div style={{opacity: reveal, transform: `translateY(${interpolate(reveal, [0,1], [24,0])}px)`}}>
      <Kicker>{scene.kicker || 'HIMIND AGENT · AUTOMEDIA WORKBENCH'}</Kicker>
      <div style={{marginTop: 30, fontSize: 130, fontWeight: 850, letterSpacing: '-.055em'}}>{scene.title || 'HiMind'}</div>
      <div style={{marginTop: 24, fontSize: 40, color: C.text, opacity: sub}}>{scene.subtitle || '让 AI 和你一起，把作品做完'}</div>
      <div style={{marginTop: 52, display: 'inline-flex', alignItems: 'center', gap: 14, padding: '17px 30px', borderRadius: 999, background: C.accent, color: '#04231f', fontSize: 24, fontWeight: 800, letterSpacing: '.02em'}}>{scene.cta || '体验 HiMind 创作台'} <span>→</span></div>
    </div>
  </AbsoluteFill>;
};

const DEFAULT_SCENES = [
  {kind: 'hero', duration_seconds: 5, title: 'HiMind', subtitle: '让 AI 成为你的创作团队'},
  {kind: 'statement', duration_seconds: 5, title: '从一句话，到一条完整的片子', subtitle: '调研 · 脚本 · 分镜 · 素材 · 剪辑 · 渲染', points: ['CreativePackage', 'Plan', 'Timeline']},
  {kind: 'screenshot', duration_seconds: 7, image: 'creative.png', title: '一句话，长出完整的创作脉络', subtitle: 'Brief → 证据链 → 脚本 → 分镜 → 制作规格', bullets: ['AI 调研与脚本', '可审阅的创作脉络', '从创意直接落到项目']},
  {kind: 'screenshot', duration_seconds: 7, image: 'timeline.png', title: '想改哪里，就改哪里', subtitle: '多轨时间线 · Inspector · 确定性渲染', bullets: ['5 轨直接编辑', '选中即调参', '每次修改都有修订']},
  {kind: 'screenshot', duration_seconds: 6, image: 'review.png', title: '每一步，都有人把关', subtitle: '审查图 · 质量门禁 · 确定性检查', bullets: ['真实画面审查', '67 项质量门禁', '可追溯的产物']},
  {kind: 'cta', duration_seconds: 5, title: 'HiMind', subtitle: '让 AI 和你一起，把作品做完', cta: '体验 HiMind 创作台'},
];

const SCENE_COMPONENTS = {hero: HeroScene, statement: StatementScene, screenshot: ScreenshotScene, cta: CTAScene};

const SCENE_OVERLAP = 8;

const SceneFade = ({duration, isLast, children}) => {
  const frame = useCurrentFrame();
  const fadeIn = interpolate(frame, [0, SCENE_OVERLAP], [0, 1], {extrapolateRight: 'clamp'});
  const fadeOut = isLast ? 1 : interpolate(frame, [duration, duration + SCENE_OVERLAP], [1, 0], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'});
  return <AbsoluteFill style={{opacity: Math.min(fadeIn, fadeOut)}}>{children}</AbsoluteFill>;
};

const SceneSlot = ({from, duration, isLast, children}) => {
  const extra = isLast ? 0 : SCENE_OVERLAP;
  return <Sequence from={from} durationInFrames={duration + extra} layout="none"><SceneFade duration={duration} isLast={isLast}>{children}</SceneFade></Sequence>;
};

export const PromoVideo = (props) => {
  const {fps} = useVideoConfig();
  const data = props.data || {};
  const scenes = Array.isArray(data.scenes) && data.scenes.length ? data.scenes : DEFAULT_SCENES;
  let cursor = 0;
  return <AbsoluteFill data-promo-root="true" style={{background: C.bg}}>
    {scenes.map((scene, index) => {
      const seconds = Math.max(2, Number(scene.duration_seconds) || 5);
      const duration = Math.round(seconds * fps);
      const from = cursor;
      cursor += duration;
      const Component = SCENE_COMPONENTS[scene.kind] || StatementScene;
      return <SceneSlot key={`${scene.kind}-${index}`} from={from} duration={duration} isLast={index === scenes.length - 1}><Component scene={scene} durationFrames={duration} /></SceneSlot>;
    })}
    <Grain />
  </AbsoluteFill>;
};

export const promoMetadata = ({props}) => {
  const canvas = props.canvas || {};
  const fps = Math.max(1, Math.min(60, Number(canvas.fps) || 30));
  const data = props.data || {};
  const scenes = Array.isArray(data.scenes) && data.scenes.length ? data.scenes : DEFAULT_SCENES;
  const total = scenes.reduce((acc, scene) => acc + Math.max(2, Number(scene.duration_seconds) || 5), 0);
  return {width: Math.max(240, Math.min(4096, Number(canvas.width) || 1920)), height: Math.max(240, Math.min(4096, Number(canvas.height) || 1080)), fps, durationInFrames: Math.max(1, Math.round(total * fps))};
};
