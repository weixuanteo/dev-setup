## Dev Setup

Quick dev setup for fresh dev environments

## Usage

Build the CLI:

```bash
go build ./cmd/devsetup
```

List the scripts that would run on macOS:

```bash
./devsetup list --os macos
```

Run the macOS scripts without executing them:

```bash
./devsetup run --os macos --dry-run
```

List the scripts that would run on Omarchy or another Arch-based Linux setup:

```bash
./devsetup list --os linux
```

Run the Linux scripts:

```bash
./devsetup run --os linux
```

## Claude skills

Interactively choose which skills from `claude/skills` to symlink into
`~/.claude/skills`:

```bash
./claude/install-skills
```

Use the arrow keys to move, Space to toggle a skill, and Enter to install. The
installer falls back to a numbered prompt when its input is redirected, and it
will not overwrite an existing skill.

## T3 Code remote server

Install a `t3code` management command on both configured SSH hosts:

```bash
./install-t3code
```

With no arguments, the installer targets the `mac-mini` and `omarchy-desktop`
aliases from `~/.ssh/config`. You can pass different aliases or `user@host`
targets:

```bash
./install-t3code server-one user@server-two
```

The remote command manages a per-user background service through launchd on
macOS or systemd on Linux:

```bash
ssh mac-mini ~/.local/bin/t3code status
ssh mac-mini ~/.local/bin/t3code restart
ssh omarchy-desktop ~/.local/bin/t3code update
ssh omarchy-desktop ~/.local/bin/t3code logs -f
```

Create a one-time remote pairing link using the server's configured Tailscale
HTTPS URL:

```bash
ssh mac-mini ~/.local/bin/t3code pair --ttl 15m --label laptop
ssh omarchy-desktop ~/.local/bin/t3code pair list
ssh omarchy-desktop ~/.local/bin/t3code pair revoke PAIRING_ID
```

Pairing links contain credentials. Avoid pasting them into logs or shared
shell history, and revoke links that are no longer needed.

List or revoke the established client sessions created after pairing:

```bash
ssh mac-mini ~/.local/bin/t3code session list
ssh omarchy-desktop ~/.local/bin/t3code session list --json
ssh omarchy-desktop ~/.local/bin/t3code session revoke SESSION_ID
```

Session listings never reveal bearer tokens. An active session is an
authorized client and is not necessarily a client with a live connection at
that exact moment.

After opening an interactive SSH session, use `t3code` directly when
`~/.local/bin` is on that machine's `PATH`; otherwise use the full path shown
above.

`update` resolves the current `t3@nightly` dist-tag, installs that exact
version, and restarts the service only when it was already running. Package
data is kept under `~/.local/share/t3code`. On Linux, installation also tries
to enable systemd user lingering so the server starts at boot and survives SSH
logout; if policy blocks that operation, the command prints the one required
`sudo loginctl` command.

Per-host configuration is stored in `~/.config/t3code/env`. The defaults bind
to `127.0.0.1:3773`, which is safe to use through an SSH tunnel:

```bash
ssh -L 3773:127.0.0.1:3773 mac-mini
```

To deploy shared settings, copy `t3code.env.example`, edit it, then run:

```bash
./install-t3code --env ./my-t3code.env
```

For direct Tailnet HTTPS access with automatic fallback, set
`T3CODE_TAILSCALE_SERVE="auto"` in the environment file. Use `"true"` to
require Tailscale or `"false"` to always bind directly to `T3CODE_HOST`. T3
Code requires Node.js `^22.16`, `^23.11`, or `>=24.10`; the selected Node and
npm must also be visible to non-interactive SSH sessions.
