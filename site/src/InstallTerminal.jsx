/* global React, CopyChip */

// ============================================================
// INSTALL — go install / binary + first-run steps
// ============================================================
function InstallTerminal() {
  const meta = window.TIHOLE_META || {};
  const goLine = (meta.install && meta.install.goLine) ||
    "go install github.com/z19r/tihole/cmd/tihole@latest";
  const configLine = (meta.install && meta.install.configLine) ||
    "~/.config/tihole/config.yaml";

  return (
    <section className="section ws-wrap" id="install">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">02 · INSTALL</div>
      </div>

      <div className="install-grid">
        <div className="steps">
          <div className="step">
            <div className="n">1</div>
            <div>
              <div className="stitle">Install with Go</div>
              <div className="sdesc">
                Grab the latest tagged release straight from the module path.
                Prefer a prebuilt binary? Download it from the GitHub releases page.
              </div>
              <div style={{ marginTop: "var(--s-3)" }}>
                <CopyChip cmd={goLine} event="install-copy-go" />
              </div>
            </div>
          </div>

          <div className="step">
            <div className="n">2</div>
            <div>
              <div className="stitle">Point it at your Pi-hole</div>
              <div className="sdesc">
                On first run tihole writes <code>{configLine}</code>. Add your
                Pi-hole v6 host and app password (or export
                <code>TIHOLE_URL</code> / <code>TIHOLE_PASSWORD</code>).
              </div>
            </div>
          </div>

          <div className="step">
            <div className="n">3</div>
            <div>
              <div className="stitle">Launch</div>
              <div className="sdesc">
                Run <code>tihole</code>. The dashboard boots straight into Gloss.
                Press <code>?</code> any time for the full key map.
              </div>
              <div style={{ marginTop: "var(--s-3)" }}>
                <CopyChip cmd="tihole" prompt="$" event="install-copy-run" />
              </div>
            </div>
          </div>
        </div>

        <div className="ws-code" aria-label="First run session">
          <div className="ws-code-head">
            <span className="ws-dots"><span className="ws-dot"></span><span className="ws-dot"></span><span className="ws-dot"></span></span>
            <span>first-run.sh</span>
          </div>
          <div className="ws-code-body">
            <span className="p">{"  $ "}</span>
            <span>{"go install github.com/z19r/tihole/cmd/tihole@latest\n"}</span>
            <span className="c">{"  // fetching · building · installed → ~/go/bin/tihole\n\n"}</span>
            <span className="p">{"  $ "}</span>
            <span>{"tihole\n"}</span>
            <span className="c">{"  // no config found — writing " + configLine + "\n"}</span>
            <span className="c">{"  // host?   "}</span>
            <span className="s">{"pi.hole\n"}</span>
            <span className="c">{"  // token?  "}</span>
            <span className="s">{"••••••••••••\n\n"}</span>
            <span className="ok">{"  ✓ connected to Pi-hole v6 (FTL 6.0)\n"}</span>
            <span className="ok">{"  ✓ gravity 148,214 domains · blocking ON\n"}</span>
            <span className="c">{"  // launching Gloss…"}</span>
          </div>
        </div>
      </div>
    </section>
  );
}

Object.assign(window, { InstallTerminal });
