/* global React */

// ============================================================
// FAQ — accordion
// ============================================================
function FAQItem({ n, q, children, open, onToggle }) {
  return (
    <div className={"faq-item" + (open ? " is-open" : "")}>
      <button className="faq-q" aria-expanded={open} onClick={onToggle} data-umami-event={"faq-" + String(n).padStart(2, "0")}>
        <span className="num">{String(n).padStart(2, "0")}</span>
        <span>{q}</span>
        <span className="glyph">{open ? "−" : "+"}</span>
      </button>
      <div className="faq-a">{children}</div>
    </div>
  );
}

function FAQ() {
  const [open, setOpen] = React.useState(0);
  const items = [
    {
      q: "Does tihole need Pi-hole v6?",
      a: (
        <React.Fragment>
          Yes. tihole speaks the Pi-hole <strong>v6</strong> REST API and session
          auth exclusively — it does not support the legacy v5 API. Point it at any
          reachable v6 instance over HTTP(S).
        </React.Fragment>
      ),
    },
    {
      q: "How do I authenticate?",
      a: (
        <React.Fragment>
          Use your Pi-hole app password. Put the host and token in{" "}
          <code>~/.config/tihole/config.yaml</code>, or export{" "}
          <code>TIHOLE_URL</code> and <code>TIHOLE_PASSWORD</code>. Nothing is
          written anywhere else and nothing is sent off-box.
        </React.Fragment>
      ),
    },
    {
      q: "What does “allow/block with one key” actually do?",
      a: (
        <React.Fragment>
          In the query log, <code>a</code> creates an exact allow rule for the
          highlighted domain and <code>b</code> creates an exact block rule — the
          same calls the web UI makes, just without the clicks. Regex and
          group-scoped rules live on the Domains screen.
        </React.Fragment>
      ),
    },
    {
      q: "Can I run it over SSH?",
      a: (
        <React.Fragment>
          That is the point. It is a single static Go binary with no daemon and no
          GUI — drop it on any box, run <code>tihole</code>, and drive your Pi-hole
          from the same terminal you already SSH into.
        </React.Fragment>
      ),
    },
    {
      q: "Does it phone home?",
      a: (
        <React.Fragment>
          No. There is zero telemetry. The only network traffic tihole makes is to
          the Pi-hole host you configure.
        </React.Fragment>
      ),
    },
    {
      q: "Can I change the theme?",
      a: (
        <React.Fragment>
          <strong>Gloss</strong> — the hot-pink-on-void look you see here — is the
          default, but tihole ships several themes. Cycle them live with{" "}
          <code>ctrl+t</code>; your choice persists.
        </React.Fragment>
      ),
    },
  ];

  return (
    <section className="section ws-wrap" id="faq">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">06 · FAQ</div>
      </div>
      <div className="faq-list">
        {items.map((it, i) => (
          <FAQItem
            key={i}
            n={i + 1}
            q={it.q}
            open={open === i}
            onToggle={() => setOpen(open === i ? -1 : i)}
          >
            {it.a}
          </FAQItem>
        ))}
      </div>
    </section>
  );
}

Object.assign(window, { FAQ });
