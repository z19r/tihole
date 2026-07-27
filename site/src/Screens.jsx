/* global React */

// ============================================================
// SCREENS — tabbed showcase of REAL tihole captures
// These are genuine screenshots of the running TUI against a live
// Pi-hole (captured with vhs), not hand-built mockups.
// ============================================================
function Shot({ label, img, alt }) {
  return (
    <div className="tui">
      <div className="tui-bar">
        <div className="dots"><i></i><i></i><i></i></div>
        <div className="title">tihole · <b>{label}</b> · live capture</div>
        <div className="badge">real</div>
      </div>
      <img
        className="shot-img"
        src={img}
        alt={alt}
        width="1440"
        height="880"
        loading="lazy"
        decoding="async"
      />
    </div>
  );
}

function Screens() {
  const tabs = [
    {
      id: "dash",
      label: "DASHBOARD",
      img: "img/screens/dashboard.webp",
      alt: "tihole dashboard: total queries, block rate, a CPU/memory/temp/health strip, a queries-over-time sparkline, and top domains/clients.",
      cap: <React.Fragment>Live totals, block-rate and a <b>host-health strip</b> — CPU, memory and temperature gauges plus FTL diagnostics.</React.Fragment>,
    },
    {
      id: "log",
      label: "QUERY LOG",
      img: "img/screens/querylog.webp",
      alt: "tihole query log: a live stream of DNS queries with time, domain, client, type and status, and a=allow / b=block hints.",
      cap: <React.Fragment>The stream you actually watch. Highlight a row, hit <b>a</b> to allow or <b>b</b> to block — no forms, no page loads.</React.Fragment>,
    },
    {
      id: "block",
      label: "BLOCKING",
      img: "img/screens/blocking.webp",
      alt: "tihole blocking pane: blocking-is-on status with enable and timed-disable actions.",
      cap: <React.Fragment>Blocking is a first-class pane. Disable with a <b>timer</b>, re-enable instantly, refresh gravity in place.</React.Fragment>,
    },
    {
      id: "dom",
      label: "DOMAINS",
      img: "img/screens/domains.webp",
      alt: "tihole domains screen: exact and regex allow/block rules with type, kind, comment, enabled and group columns.",
      cap: <React.Fragment>Full CRUD over exact and <b>regex</b> allow/block rules, scoped to groups — same keys everywhere.</React.Fragment>,
    },
  ];
  const [active, setActive] = React.useState("dash");
  const current = tabs.find((t) => t.id === active) || tabs[0];

  return (
    <section className="section ws-wrap" id="screens">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">01 · SCREENS</div>
        <p className="ws-sec-sub">Real captures from a live Pi-hole — not mockups.</p>
      </div>

      <div className="screens-tabs" role="tablist" aria-label="Screens">
        {tabs.map((t, i) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={active === t.id}
            className={"screens-tab" + (active === t.id ? " is-active" : "")}
            onClick={() => setActive(t.id)}
            data-umami-event={"screens-tab-" + t.id}
          >
            <span className="ix">{String(i + 1).padStart(2, "0")}</span>
            {t.label}
          </button>
        ))}
      </div>

      <div className="screens-stage">
        <div className="glow royal" style={{ width: 320, height: 320, bottom: -80, left: "30%" }}></div>
        <Shot label={current.label} img={current.img} alt={current.alt} />
      </div>

      <p className="screens-cap">{current.cap}</p>
    </section>
  );
}

Object.assign(window, { Screens });
