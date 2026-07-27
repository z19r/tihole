# Troubleshooting

tihole is designed to fail *soft*: a wrong address or password never aborts
startup. Authentication happens lazily, so connection problems show up as in-app
error banners, and a structurally broken config drops you straight into the
config editor to fix it. Most issues below are diagnosed and fixed without ever
leaving the app.

## Config file

Config lives at `~/.config/tihole/config.yaml` (more precisely
`<UserConfigDir>/tihole/config.yaml`), created mode `0600` in a `0700`
directory. It's written and edited by the app, but you can hand-author it — see
the [README](../README.md#configuration) for the full schema.

Open the editor any time with `tihole config`, or from the command palette
(`ctrl+k` → Settings → Connections).

### "no instances configured" / "active does not match any instance"

The config failed structural validation. tihole checks that:

- at least one instance is configured,
- every instance has a non-empty, **unique** `name`,
- every instance `url` parses as an absolute `http`/`https` URL with a host,
- `active` names exactly one existing instance.

All problems are reported together. Fix them in the config editor (`tihole
config`) — it opens automatically when validation fails.

### "instance has no password or password_env" / "password_env is empty or unset"

An instance needs an app password supplied one of two ways:

```yaml
instances:
  - name: home
    url: https://pi.hole
    password_env: TIHOLE_HOME_PASSWORD   # preferred: read from the environment
  - name: cabin
    url: http://10.0.0.53
    password: inline-ok-but-env-is-better
```

If you use `password_env`, make sure the named variable is actually exported in
the environment tihole runs in (`export TIHOLE_HOME_PASSWORD=…`). An unset or
empty variable produces the "empty or unset" error.

## Connection & authentication

### Login fails / "authentication" banner

tihole authenticates lazily using Pi-hole v6's `X-FTL-SID` session header, and
re-authenticates transparently when a session expires. A persistent auth banner
almost always means the **app password is wrong** or points at the wrong
instance. Check:

- The password (or the value of `password_env`) matches the Pi-hole admin
  **app password**, not your web-UI login if they differ.
- `url` points at the right host and scheme.

Passwords and session IDs are never logged, so you won't find them in any output
— that's by design.

### TLS certificate errors (self-signed Pi-hole)

If your Pi-hole uses a self-signed certificate or no TLS, set `verify_tls: false`
on that instance:

```yaml
  - name: cabin
    url: https://10.0.0.53
    verify_tls: false
```

`verify_tls` defaults to `true` when omitted. Only disable it for instances you
control on a trusted network.

## Destructive actions rejected (403)

Some actions — **restart DNS**, **flush logs**, **flush network table** — are
destructive and require `webserver.api.allow_destructive` to be enabled on the
Pi-hole server. Without it, the API returns `403` and tihole surfaces a clear
hint. Enable the setting in the Pi-hole admin (Settings → System / API) or via
the FTL config, then retry.

## Display & rendering

### Misaligned sidebar / broken box drawing

tihole uses single-cell BMP glyphs for its sidebar icons specifically to avoid
the double-width emoji-presentation problem. If labels still look misaligned,
the culprit is usually the **terminal font or an ambiguous-width setting** —
ensure a font with proper box-drawing coverage and that ambiguous-width
characters are treated as single-width.

### Colors look wrong / theme not applying

- Cycle themes with `ctrl+t` or pick one from the palette (`ctrl+k`) to confirm
  theming works at all.
- The `omarchy` theme reads `~/.config/omarchy/current/theme/alacritty.toml`; if
  that file is missing or malformed, tihole falls back to `deep-night` rather
  than erroring.
- Full-color themes need a true-color (24-bit) terminal. In a 256-color terminal
  gradients degrade; set `COLORTERM=truecolor` if your terminal supports it but
  doesn't advertise it.

## Still stuck?

- `tihole help` — usage.
- [`docs/keys.md`](keys.md) — full key reference.
- [`docs/architecture.md`](architecture.md) — how the pieces fit together.
- Open an issue at <https://github.com/z19r/tihole/issues>.
