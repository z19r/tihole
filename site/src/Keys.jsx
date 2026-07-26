/* global React */

// ============================================================
// KEYS — the shortcut map, keycap-styled
// ============================================================
function Cap({ children, tone }) {
  return <span className={"kbd" + (tone ? " " + tone : "")}>{children}</span>;
}

function KeyRow({ caps, act, ctx }) {
  return (
    <div className="keyrow">
      <span style={{ display: "inline-flex", gap: 6, alignItems: "center" }}>
        {caps.map((c, i) => (
          <React.Fragment key={i}>
            {i > 0 && <span className="key-plus">+</span>}
            <Cap tone={c.tone}>{c.k}</Cap>
          </React.Fragment>
        ))}
      </span>
      <span className="kmeta">
        <span className="act">{act}</span>
        <span className="ctx">{ctx}</span>
      </span>
    </div>
  );
}

function Keys() {
  const rows = [
    { caps: [{ k: "j" }, { k: "k" }], act: "Move down / up", ctx: "any list" },
    { caps: [{ k: "a", tone: "allow" }], act: "Allow domain", ctx: "query log" },
    { caps: [{ k: "b", tone: "block" }], act: "Block domain", ctx: "query log" },
    { caps: [{ k: "/", tone: "hot" }], act: "Search / filter", ctx: "any screen" },
    { caps: [{ k: "tab" }], act: "Next section", ctx: "global" },
    { caps: [{ k: "n" }], act: "New record", ctx: "domains · dns · groups" },
    { caps: [{ k: "e" }], act: "Edit record", ctx: "any CRUD list" },
    { caps: [{ k: "x", tone: "block" }], act: "Delete record", ctx: "any CRUD list" },
    { caps: [{ k: "d", tone: "hot" }], act: "Disable blocking", ctx: "with timer" },
    { caps: [{ k: "r" }], act: "Refresh / reload", ctx: "global" },
    { caps: [{ k: "ctrl" }, { k: "t", tone: "acid" }], act: "Cycle theme", ctx: "Gloss + more" },
    { caps: [{ k: "?", tone: "acid" }], act: "Help / key map", ctx: "global" },
  ];

  return (
    <section className="section ws-wrap" id="keys">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">05 · KEYS</div>
      </div>
      <div className="keys-grid">
        {rows.map((r, i) => (
          <KeyRow key={i} caps={r.caps} act={r.act} ctx={r.ctx} />
        ))}
      </div>
    </section>
  );
}

Object.assign(window, { Keys });
