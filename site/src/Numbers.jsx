/* global React */

// ============================================================
// NUMBERS — the stats rail
// ============================================================
function Numbers() {
  const stats = [
    { ix: "SURFACES", num: "10", u: "screens", cls: "acid", sub: "dashboard → local DNS" },
    { ix: "KEYSTROKE", num: "1", u: "key", cls: "mag", sub: "allow / block a domain" },
    { ix: "FOOTPRINT", num: "0", u: "runtime deps", cls: "", sub: "one static Go binary" },
    { ix: "TELEMETRY", num: "0", u: "phone-homes", cls: "acid", sub: "nothing leaves your box" },
  ];
  return (
    <section className="section ws-wrap" id="numbers">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">03 · NUMBERS</div>
      </div>
      <div className="stat-rail-x">
        {stats.map((s) => (
          <div className="stat-x" key={s.ix}>
            <div className="ix">{s.ix}</div>
            <div className={"num " + s.cls}>
              {s.num}<span className="u">{s.u}</span>
            </div>
            <div className="sub">{s.sub}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

Object.assign(window, { Numbers });
