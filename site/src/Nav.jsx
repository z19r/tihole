/* global React */

// ============================================================
// NAV — sticky top with theme toggle
// ============================================================
function THMark({ size = 32 }) {
  // A blocked-request glyph — the ⦸ motif tihole uses for its Blocking pane.
  return (
    <svg width={size} height={size} viewBox="0 0 200 200" fill="none" aria-hidden="true">
      <rect x="6" y="6" width="188" height="188" fill="var(--mark-bg, #110628)" stroke="var(--mark-fg, #F5F2E8)" strokeWidth="4" />
      <circle cx="100" cy="100" r="52" stroke="var(--mark-fg, #F5F2E8)" strokeWidth="14" fill="none" />
      <path d="M63 63 L137 137" stroke="#D6FF2E" strokeWidth="14" strokeLinecap="square" />
      <rect x="40" y="160" width="120" height="4" fill="#FF2E93" />
    </svg>
  );
}

function Wordmark({ size = 22 }) {
  return (
    <span className="ws-wordmark" style={{ fontSize: size }}>
      <span className="s1">/</span><span>/TI</span><span className="dot">·</span><span>HOLE</span>
    </span>
  );
}

function Nav({ theme, onToggleTheme }) {
  const sections = [
    { id: 'screens',  label: '01 · SCREENS' },
    { id: 'install',  label: '02 · INSTALL' },
    { id: 'numbers',  label: '03 · NUMBERS' },
    { id: 'keys',     label: '04 · KEYS' },
    { id: 'releases', label: '05 · RELEASES' },
    { id: 'faq',      label: '06 · FAQ' },
  ];

  const isLight = theme === 'light';

  return (
    <nav className="ws-nav">
      <a href="#top" className="ws-nav-mark" data-umami-event="nav-logo" style={{
        '--mark-bg': isLight ? '#F5F2E8' : '#110628',
        '--mark-fg': isLight ? '#07030E' : '#F5F2E8',
      }}>
        <THMark size={32} />
        <Wordmark size={20} />
      </a>

      <div className="ws-nav-links">
        {sections.map((s) => (
          <a key={s.id} href={'#' + s.id} className="ws-nav-link" data-umami-event={"nav-" + s.id}>{s.label}</a>
        ))}
      </div>

      <div className="ws-nav-right">
        <span className="ws-badge ws-badge--acid"><span className="pulse"></span>v{window.TIHOLE_VERSION}</span>
        <button className="theme-toggle" onClick={onToggleTheme} aria-label="Toggle theme" data-umami-event="theme-toggle">
          <span className="dot"></span>
          {isLight ? 'LIGHT' : 'NIGHT'}
        </button>
        <a className="ws-btn ws-btn--sm ws-btn--ghost" href="https://github.com/z19r/tihole" target="_blank" rel="noreferrer" data-umami-event="nav-github">GITHUB ↗</a>
      </div>
    </nav>
  );
}

Object.assign(window, { Nav, THMark, Wordmark });
