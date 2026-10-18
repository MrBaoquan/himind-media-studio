// foundation/background.jsx —— 所有镜头共用的底。
export function Background({palette, px}) {
  return (
    <>
      <div
        style={{
          position: 'absolute',
          inset: 0,
          background: `linear-gradient(180deg, ${palette.bg0} 0%, ${palette.bg1} 42%, ${palette.bg2} 74%, ${palette.bg0} 100%)`,
        }}
      />
      <div
        style={{
          position: 'absolute',
          top: -px(240),
          left: '50%',
          transform: 'translateX(-50%)',
          width: px(1500),
          height: px(900),
          background: `radial-gradient(ellipse, ${palette.accent}22, transparent 70%)`,
          filter: `blur(${px(60)}px)`,
        }}
      />
      <div
        style={{
          position: 'absolute',
          inset: 0,
          backgroundImage: `radial-gradient(circle at 1px 1px, ${palette.line} 1px, transparent 0)`,
          backgroundSize: `${px(54)}px ${px(54)}px`,
          opacity: 0.7,
        }}
      />
    </>
  );
}
