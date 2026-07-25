/* global React */

// ============================================================
// ARCHITECTURE — how tihole talks to Pi-hole v6
// ============================================================
function Architecture() {
  return (
    <section className="section ws-wrap" id="architecture">
      <div className="ws-sec-head">
        <div className="ws-sec-tag">04 · ARCHITECTURE</div>
        <h2>A thin, honest client.</h2>
      </div>

      <div className="arch">
        <div className="col">
          <div className="node user">
            <span className="tag">YOU</span>
            <span className="name">Terminal</span>
            <span className="desc">Any shell over SSH or local. Keyboard in, Gloss out.</span>
          </div>
          <div className="node api">
            <span className="tag">PI-HOLE V6</span>
            <span className="name">FTL · REST</span>
            <span className="desc">The official v6 HTTP API + session auth. tihole never touches the DB directly.</span>
          </div>
        </div>

        <div className="right">
          <div className="rail">
            <div className="lane rtk">
              <div>
                <div className="ltag">TUI LAYER</div>
                <div className="desc">Bubble Tea model/update/view · lipgloss “Gloss” theme.</div>
              </div>
              <span className="gain">render</span>
            </div>
            <div className="lane hr">
              <div>
                <div className="ltag">DOMAIN</div>
                <div className="desc">Blocking · query log · domains · groups · clients · adlists · local DNS.</div>
              </div>
              <span className="gain">state</span>
            </div>
            <div className="lane mem">
              <div>
                <div className="ltag">API CLIENT</div>
                <div className="desc">Typed Pi-hole v6 client · auth, retries, polling.</div>
              </div>
              <span className="gain">http</span>
            </div>
          </div>

          <div className="arrow-col">
            <span className="arrow">→</span>
            <span>keys</span>
            <span className="arrow">←</span>
            <span>json</span>
          </div>

          <div className="rail">
            <div className="lane">
              <div>
                <div className="ltag">CONFIG</div>
                <div className="desc">~/.config/tihole/config.yaml · env overrides.</div>
              </div>
              <span className="gain">yaml</span>
            </div>
            <div className="lane">
              <div>
                <div className="ltag">THEMES</div>
                <div className="desc">Gloss default · swap live with ctrl+t.</div>
              </div>
              <span className="gain">live</span>
            </div>
            <div className="lane">
              <div>
                <div className="ltag">SHIP</div>
                <div className="desc">One static Go binary · no daemon, no telemetry.</div>
              </div>
              <span className="gain">go</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

Object.assign(window, { Architecture });
