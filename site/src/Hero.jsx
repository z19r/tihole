/* global React */

// ============================================================
// HERO — gradient wordmark, one-key value prop, live TUI preview
// ============================================================
function CopyChip({ cmd, prompt = "$", event = "copy-command" }) {
  const [copied, setCopied] = React.useState(false);
  const onCopy = () => {
    try {
      navigator.clipboard.writeText(cmd);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch (_) {}
  };
  return (
    <div className="copy-chip">
      <span className="prompt">{prompt}</span>
      <span className="cmd">{cmd}</span>
      <button
        className={"copy-btn" + (copied ? " is-copied" : "")}
        onClick={onCopy}
        aria-label="Copy install command"
        data-umami-event={event}
        data-umami-event-cmd={cmd}
      >
        {copied ? "COPIED" : "COPY"}
      </button>
    </div>
  );
}

function DashboardPreview() {
  return;
  return (
    <div className="tui" role="img" aria-label="tihole dashboard preview">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>DASHBOARD</b></div>
        <div className="badge">gloss</div>
      </div>
      <div className="tui-strip">
        <span className="tui-pill on">● BLOCKING ON</span>
        <span className="tui-pill dim">FTL 6.0 · UP 12d</span>
        <span className="tui-pill dim">GRAVITY 148,214</span>
      </div>
      <div className="tui-body">
        <div style={{ display: "grid", gap: "8px" }}>
          <div className="tui-gauge">
            <span className="lab">Queries</span>
            <span className="track"><span className="fill" style={{ right: "26%" }}></span></span>
            <span className="val">74,391</span>
          </div>
          <div className="tui-gauge">
            <span className="lab">Blocked</span>
            <span className="track"><span className="fill" style={{ right: "77%" }}></span></span>
            <span className="val">23.1%</span>
          </div>
          <div className="tui-gauge">
            <span className="lab">CPU</span>
            <span className="track"><span className="fill" style={{ right: "82%" }}></span></span>
            <span className="val">18%</span>
          </div>
          <div className="tui-gauge">
            <span className="lab">Memory</span>
            <span className="track"><span className="fill" style={{ right: "59%" }}></span></span>
            <span className="val">41%</span>
          </div>
          <div className="tui-gauge">
            <span className="lab">Temp</span>
            <span className="track"><span className="fill" style={{ right: "63%" }}></span></span>
            <span className="val">47°C</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function Hero() {
  const meta = window.TIHOLE_META || {};
  const goLine = (meta.install && meta.install.goLine) ||
    "go install github.com/z19r/tihole/cmd/tihole@latest";

  return (
    <header className="hero ws-wrap" id="top">
      <div className="glow pink drift" style={{ width: 460, height: 460, top: -140, left: -120 }}></div>
      <div className="glow cyan drift" style={{ width: 380, height: 380, top: 40, right: -100 }}></div>

      <div style={{ position: "relative", zIndex: 1, display: "grid", gridTemplateColumns: "minmax(0,1fr)", gap: "var(--s-6)" }}>
        <div>
          <div className="eyebrow-row">
            <span className="bar"></span>
          </div>

          <h1 className="display">
            Run Pi-hole from<br />
            the <span className="ramp">terminal.</span>
          </h1>

          <p className="sub">
            <b>tihole</b> is a fast, keyboard-driven TUI for Pi-hole v6. Watch the
            live query log and <b>allow or block a domain with a single key</b> — then
            manage blocking, groups, clients, adlists and local DNS without ever
            leaving the shell.
          </p>

          <div className="hero-cta">
            <CopyChip cmd={goLine} event="hero-copy-install" />
            <a className="ws-btn ws-btn--ghost" href="https://github.com/z19r/tihole" target="_blank" rel="noreferrer" data-umami-event="hero-github-star">
              STAR ON GITHUB ↗
            </a>
          </div>

        </div>

        <DashboardPreview />
      </div>
    </header>
  );
}

Object.assign(window, { Hero, CopyChip, DashboardPreview });
