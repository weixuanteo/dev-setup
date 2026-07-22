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
