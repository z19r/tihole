/* global React, Nav, Hero, Screens, InstallTerminal, Numbers, Architecture, Keys, Releases, FAQ, DocsLinks, Footer */

// ============================================================
// APP — top-level composition
// ============================================================
function App() {
  // theme persistence — sync to <html data-theme="…">
  // Priority: explicit user choice (localStorage) > OS preference (default dark).
  const getSystemTheme = () =>
    (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches)
      ? 'light' : 'dark';

  const [theme, setTheme] = React.useState(() => {
    try {
      const saved = localStorage.getItem('th-theme');
      if (saved === 'light' || saved === 'dark') return saved;
    } catch (_) {}
    return getSystemTheme();
  });
  const [hasExplicitChoice, setHasExplicitChoice] = React.useState(() => {
    try { return !!localStorage.getItem('th-theme'); } catch (_) { return false; }
  });

  // Apply theme to <html>
  React.useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  // Track OS preference changes — follow them unless the user has chosen explicitly
  React.useEffect(() => {
    if (hasExplicitChoice) return;
    const mq = window.matchMedia && window.matchMedia('(prefers-color-scheme: light)');
    if (!mq) return;
    const onChange = (e) => setTheme(e.matches ? 'light' : 'dark');
    mq.addEventListener ? mq.addEventListener('change', onChange) : mq.addListener(onChange);
    return () => {
      mq.removeEventListener ? mq.removeEventListener('change', onChange) : mq.removeListener(onChange);
    };
  }, [hasExplicitChoice]);

  const choose = (next) => {
    setTheme(next);
    setHasExplicitChoice(true);
    try { localStorage.setItem('th-theme', next); } catch (_) {}
  };

  const onToggleTheme = () => choose(theme === 'light' ? 'dark' : 'light');

  return (
    <React.Fragment>
      <div className="page-grid"></div>

      <Nav theme={theme} onToggleTheme={onToggleTheme} />

      <main>
        <Hero />

        <div className="marquee" aria-hidden="true">
          <div className="marquee-track">
            <span>PI-HOLE V6 <span className="sep">·</span></span>
            <span>GO 1.23 <span className="sep">·</span></span>
            <span>10 SCREENS <span className="sep">·</span></span>
            <span>ONE BINARY <span className="sep">·</span></span>
            <span>KEYBOARD-DRIVEN <span className="sep">·</span></span>
            <span>NO TELEMETRY <span className="sep">·</span></span>
            <span>v{window.TIHOLE_VERSION} <span className="sep">·</span></span>
            {/* duplicate for seamless loop */}
            <span>PI-HOLE V6 <span className="sep">·</span></span>
            <span>GO 1.23 <span className="sep">·</span></span>
            <span>10 SCREENS <span className="sep">·</span></span>
            <span>ONE BINARY <span className="sep">·</span></span>
            <span>KEYBOARD-DRIVEN <span className="sep">·</span></span>
            <span>NO TELEMETRY <span className="sep">·</span></span>
            <span>v{window.TIHOLE_VERSION} <span className="sep">·</span></span>
          </div>
        </div>

        <Screens />
        <InstallTerminal />
        <Numbers />
        <Architecture />
        <Keys />
        <Releases />
        <FAQ />
        <DocsLinks />
      </main>

      <Footer />
    </React.Fragment>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
