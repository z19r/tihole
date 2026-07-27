/* global React, Wordmark */

// ============================================================
// FOOTER
// ============================================================
function Footer() {
  const meta = window.TIHOLE_META || {};
  const repo = "https://github.com/z19r/tihole";
  return (
    <footer className="footer">
      <div className="ws-wrap">
        <div className="row">
          <div className="footer-mark">
            <span className="grad-ramp" style={{ fontFamily: "var(--f-sans)", fontWeight: 700, fontSize: 30, letterSpacing: "-0.03em" }}>
              //TI·HOLE
            </span>
            <p>
              A fast, keyboard-driven terminal UI for Pi-hole v6. Watch the query
              log, allow or block in one key, and manage everything else without
              leaving the shell.
            </p>
            <div className="hero-badges">
              <span className="ws-badge ws-badge--mag">v{window.TIHOLE_VERSION}</span>
              <span className="ws-badge ws-badge--ghost">MIT</span>
            </div>
          </div>

          <div className="col">
            <h4>Project</h4>
            <ul>
              <li><a href={repo} target="_blank" rel="noreferrer" data-umami-event="footer-github">GitHub</a></li>
              <li><a href={repo + "/releases"} target="_blank" rel="noreferrer" data-umami-event="footer-releases">Releases</a></li>
              <li><a href={repo + "/issues"} target="_blank" rel="noreferrer" data-umami-event="footer-issues">Issues</a></li>
              <li><a href={repo + "/blob/main/CHANGELOG.md"} target="_blank" rel="noreferrer" data-umami-event="footer-changelog">Changelog</a></li>
            </ul>
          </div>

          <div className="col">
            <h4>Navigate</h4>
            <ul>
              <li><a href="#screens" data-umami-event="footer-nav-screens">Screens</a></li>
              <li><a href="#install" data-umami-event="footer-nav-install">Install</a></li>
              <li><a href="#keys" data-umami-event="footer-nav-keys">Keys</a></li>
              <li><a href="#faq" data-umami-event="footer-nav-faq">FAQ</a></li>
            </ul>
          </div>
        </div>

        <div className="meta">
          <span>© {new Date().getFullYear()} Z19R · MIT</span>
          <span>MADE IN CHICAGO, WITH 🫀 ©2026 z19r. All rights reserved.</span>
          <span>{meta.sha ? "build " + meta.sha : "github.com/z19r/tihole"} · Pi-hole v6</span>
        </div>
      </div>

      <div className="watermark" aria-hidden="true">⦸</div>
    </footer>
  );
}

Object.assign(window, { Footer });
