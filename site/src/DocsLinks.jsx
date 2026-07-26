/* global React */

// ============================================================
// DOCS — jump-off cards
// ============================================================
function DocsLinks() {
  const repo = "https://github.com/z19r/tihole";
  const cards = [
    { eyebrow: "START HERE", title: "README", desc: "Install, configure, and connect to your first Pi-hole v6 host.", more: "READ", href: repo + "#readme" },
    { eyebrow: "REFERENCE", title: "Key map", desc: "Every shortcut on every screen, printable at a glance.", more: "OPEN", href: repo + "/blob/main/docs/keys.md" },
    { eyebrow: "REFERENCE", title: "Config", desc: "config.yaml fields, env overrides, and multiple hosts.", more: "OPEN", href: repo + "/blob/main/docs/config.md" },
    { eyebrow: "SOURCE", title: "GitHub", desc: "Star it, file an issue, or read the Go that drives it.", more: "VISIT ↗", href: repo },
  ];
  return (
    <section className="section ws-wrap" id="docs">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">08 · DOCS</div>
      </div>
      <div className="docs-grid">
        {cards.map((c) => (
          <a className="doc-card" key={c.title} href={c.href} target="_blank" rel="noreferrer">
            <span className="eyebrow">{c.eyebrow}</span>
            <span className="title">{c.title}</span>
            <span className="desc">{c.desc}</span>
            <span className="more">{c.more} <span aria-hidden="true">→</span></span>
          </a>
        ))}
      </div>
    </section>
  );
}

Object.assign(window, { DocsLinks });
