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
