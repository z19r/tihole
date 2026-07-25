/* global React */

// ============================================================
// SCREENS — tabbed faux-TUI showcase of the real app surfaces
// ============================================================
function QueryRow({ t, dom, ok, sel }) {
  return (
    <div className={"qrow" + (sel ? " sel" : "")}>
      <span className="t">{t}</span>
      <span className="dom">{dom}</span>
      <span className={"st " + (ok ? "ok" : "no")}>{ok ? "ALLOWED" : "BLOCKED"}</span>
      <span className="act">
        <kbd className="a">a</kbd>
        <kbd className="b">b</kbd>
      </span>
    </div>
  );
}

function ScreenDashboard() {
  return (
    <div className="tui">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>DASHBOARD</b></div>
        <div className="badge">01</div>
      </div>
      <div className="tui-strip">
        <span className="tui-pill on">● BLOCKING ON</span>
        <span className="tui-pill dim">CLIENTS 24</span>
        <span className="tui-pill dim">ADLISTS 11</span>
      </div>
      <div className="tui-body">
        <pre>{`  QUERIES (24h)     74,391      ▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░
  BLOCKED           17,180      ▓▓▓▓░░░░░░░░░░░░   23.1%
  CACHE HIT         61.4%       ▓▓▓▓▓▓▓▓▓▓░░░░░░
  FORWARDED         28.9%       ▓▓▓▓▓░░░░░░░░░░░

  TOP BLOCKED
    · doubleclick.net            2,041
    · googlesyndication.com      1,660
    · graph.facebook.com           994`}</pre>
      </div>
    </div>
  );
}

function ScreenQueryLog() {
  return (
    <div className="tui">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>QUERY LOG</b> · press a=allow  b=block</div>
        <div className="badge">live</div>
      </div>
      <div className="tui-body" style={{ padding: 0 }}>
        <QueryRow t="18:42:07" dom="analytics.tiktok.com" ok={false} />
        <QueryRow t="18:42:06" dom="api.github.com" ok={true} />
        <QueryRow t="18:42:04" dom="ads.doubleclick.net" ok={false} sel />
        <QueryRow t="18:42:01" dom="cdn.jsdelivr.net" ok={true} />
        <QueryRow t="18:41:58" dom="telemetry.samsung.com" ok={false} />
        <QueryRow t="18:41:55" dom="raw.githubusercontent.com" ok={true} />
      </div>
    </div>
  );
}

function ScreenBlocking() {
  return (
    <div className="tui">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>BLOCKING</b></div>
        <div className="badge">⦸</div>
      </div>
      <div className="tui-body">
        <pre>{`  STATUS         ● ENABLED

  DISABLE FOR…
    [ 10s ]   [ 30s ]   [ 5m ]   [ 30m ]   [ ∞ ]

  > enter  toggle now      d  disable w/ timer
    e      re-enable       r  refresh gravity`}</pre>
      </div>
    </div>
  );
}

function ScreenDomains() {
  return (
    <div className="tui">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>DOMAINS</b> · exact + regex</div>
        <div className="badge">crud</div>
      </div>
      <div className="tui-body">
        <pre>{`  TYPE    KIND     DOMAIN                       GROUP
  ─────   ──────   ──────────────────────────   ──────
  ALLOW   exact    api.github.com               default
  BLOCK   exact    ads.doubleclick.net          default
  BLOCK   regex    (^|\\.)facebook\\.com$          kids
  ALLOW   regex    (^|\\.)apple\\.com$             default

  n  new     e  edit     x  delete     /  search`}</pre>
      </div>
    </div>
  );
}

function Screens() {
  const tabs = [
    { id: "dash",  label: "DASHBOARD", el: <ScreenDashboard />, cap: <React.Fragment>Live totals, block-rate and a <b>host-health strip</b> — CPU, memory and temperature gauges plus FTL diagnostics.</React.Fragment> },
    { id: "log",   label: "QUERY LOG", el: <ScreenQueryLog />, cap: <React.Fragment>The stream you actually watch. Hover a row, hit <b>a</b> to allow or <b>b</b> to block — no forms, no page loads.</React.Fragment> },
    { id: "block", label: "BLOCKING", el: <ScreenBlocking />, cap: <React.Fragment>Blocking is a first-class pane. Disable with a <b>timer</b>, re-enable instantly, refresh gravity in place.</React.Fragment> },
    { id: "dom",   label: "DOMAINS", el: <ScreenDomains />, cap: <React.Fragment>Full CRUD over exact and <b>regex</b> allow/block rules, scoped to groups — same keys everywhere.</React.Fragment> },
  ];
  const [active, setActive] = React.useState("dash");
  const current = tabs.find((t) => t.id === active) || tabs[0];

  return (
    <section className="section ws-wrap" id="screens">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">01 · SCREENS</div>
        <h2>Ten screens, one muscle memory.</h2>
      </div>

      <div className="screens-tabs" role="tablist" aria-label="Screens">
        {tabs.map((t, i) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={active === t.id}
            className={"screens-tab" + (active === t.id ? " is-active" : "")}
            onClick={() => setActive(t.id)}
          >
            <span className="ix">{String(i + 1).padStart(2, "0")}</span>
            {t.label}
          </button>
        ))}
      </div>

      <div className="screens-stage">
        <div className="glow royal" style={{ width: 320, height: 320, bottom: -80, left: "30%" }}></div>
        {current.el}
      </div>

      <p className="screens-cap">{current.cap}</p>
    </section>
  );
}

Object.assign(window, { Screens });
