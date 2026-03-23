# Installation

## Quick Install (Linux / macOS)

Download the latest release, verify its checksum, install the binary and shell completion helpers:

```bash
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-debug-queries/main/install.sh | bash
```

By default the script installs to `~/.local/bin`. Override with environment variables:

```bash
# Install a specific version
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-debug-queries/main/install.sh | VERSION=v0.1.0 bash

# Install to a different directory
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-debug-queries/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

The script installs three files:

| File | Purpose |
|------|---------|
| `kubectl-debug_queries` | Main binary (kubectl plugin) |
| `kubectl_complete-debug_queries` | Shell completion helper for `kubectl debug-queries` |
| `oc_complete-debug_queries` | Shell completion helper for `oc debug-queries` |

If the install directory is not in your `PATH`, the script prints instructions for adding it.

## Install with Krew

If you use [krew](https://krew.sigs.k8s.io/) (the kubectl plugin manager):

```bash
kubectl krew install debug-queries
```

To upgrade later:

```bash
kubectl krew upgrade debug-queries
```

## Build from Source

Requires Go 1.21+.

```bash
git clone https://github.com/yaacov/kubectl-debug-queries.git
cd kubectl-debug-queries
make build

# Copy the binary to a directory in your PATH (note the underscore in the target name)
sudo cp kubectl-debug-queries /usr/local/bin/kubectl-debug_queries
```

## Manual Download

Download a release archive from the [GitHub Releases](https://github.com/yaacov/kubectl-debug-queries/releases) page. Archives are available for:

| OS | Architecture | Archive |
|----|-------------|---------|
| Linux | amd64 | `kubectl-debug-queries-VERSION-linux-amd64.tar.gz` |
| Linux | arm64 | `kubectl-debug-queries-VERSION-linux-arm64.tar.gz` |
| macOS | amd64 | `kubectl-debug-queries-VERSION-darwin-amd64.tar.gz` |
| macOS | arm64 | `kubectl-debug-queries-VERSION-darwin-arm64.tar.gz` |
| Windows | amd64 | `kubectl-debug-queries-VERSION-windows-amd64.zip` |

Extract and install:

```bash
VERSION=v0.1.0   # replace with desired version
OS=darwin         # linux, darwin, or windows
ARCH=arm64        # amd64 or arm64

tar -xzf kubectl-debug-queries-${VERSION}-${OS}-${ARCH}.tar.gz
install -m 0755 kubectl-debug-queries-${OS}-${ARCH} ~/.local/bin/kubectl-debug_queries
```

## Shell Completion

Tab completion works for both `kubectl debug-queries` and `oc debug-queries`. The [install script](#quick-install-linux--macos) sets this up automatically. If you installed via another method, create the helpers manually:

```bash
# Find the directory where kubectl-debug_queries is installed
d="$(dirname "$(which kubectl-debug_queries)")"

# Create the kubectl completion helper
cat > "$d/kubectl_complete-debug_queries" << 'SCRIPT'
#!/usr/bin/env bash
kubectl-debug_queries __complete "$@"
SCRIPT
chmod +x "$d/kubectl_complete-debug_queries"

# Create the oc completion helper (symlink to the kubectl one)
ln -sf "$d/kubectl_complete-debug_queries" "$d/oc_complete-debug_queries"
```

Requires kubectl 1.26+ or oc 4.x with shell completions loaded.

## Uninstall

Remove the three installed files:

```bash
rm -f ~/.local/bin/kubectl-debug_queries
rm -f ~/.local/bin/kubectl_complete-debug_queries
rm -f ~/.local/bin/oc_complete-debug_queries
```

If you installed to a different directory, replace `~/.local/bin` with that path.
