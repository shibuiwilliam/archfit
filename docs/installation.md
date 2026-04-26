# Installation

## From source

Requires **Go 1.24+**. No CGO.

```bash
git clone https://github.com/shibuiwilliam/archfit.git
cd archfit
go build -o ./bin/archfit ./cmd/archfit
./bin/archfit version
```

## From release binaries

Pre-built binaries are published on [GitHub Releases](https://github.com/shibuiwilliam/archfit/releases) for every tagged version.

| Platform | Archive |
|----------|---------|
| Linux amd64 | `archfit-v*-linux-amd64.tar.gz` |
| Linux arm64 | `archfit-v*-linux-arm64.tar.gz` |
| macOS Intel | `archfit-v*-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `archfit-v*-darwin-arm64.tar.gz` |
| Windows amd64 | `archfit-v*-windows-amd64.zip` |

### macOS / Linux

```bash
# Download (replace version and platform as needed)
curl -sSL https://github.com/shibuiwilliam/archfit/releases/latest/download/archfit-v0.1.0-darwin-arm64.tar.gz \
  | tar xz

# Move to a directory on PATH
sudo mv archfit-* /usr/local/bin/archfit

# Verify
archfit version
```

### Windows

1. Download the `.zip` from [Releases](https://github.com/shibuiwilliam/archfit/releases)
2. Extract `archfit-v*-windows-amd64.exe`
3. Move it to a directory on your `%PATH%` (e.g., `C:\tools\`) or add its location to `PATH`

```powershell
archfit.exe version
```

## Via `go install`

If you have Go installed, this is the simplest method:

```bash
go install github.com/shibuiwilliam/archfit/cmd/archfit@latest
```

The binary is placed in `$GOPATH/bin` (usually `$HOME/go/bin`), which should already be on your `$PATH`.

## Via Docker

No local installation required:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/shibuiwilliam/archfit:latest scan /repo
```

## Adding to PATH

If `archfit version` gives "command not found" after installation, the binary isn't on your `PATH`.

### macOS / Linux

```bash
# Option A: copy to a standard location
sudo cp ./bin/archfit /usr/local/bin/archfit

# Option B: use a user-local directory (no sudo)
mkdir -p ~/.local/bin
cp ./bin/archfit ~/.local/bin/archfit

# Add to PATH (append to ~/.zshrc, ~/.bashrc, or ~/.profile)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Windows

```powershell
# Option A: copy to an existing PATH directory
copy .\bin\archfit.exe C:\Windows\archfit.exe

# Option B: create a dedicated directory and add to PATH
mkdir C:\tools
copy .\bin\archfit.exe C:\tools\archfit.exe
# Then: System > Environment Variables > PATH > Add C:\tools
```

## Verify installation

```bash
archfit version
# archfit 0.1.0

archfit scan --json . | head -5
# {"schema_version":"0.1.0","tool":{"name":"archfit",...
```

## Next steps

- [Getting Started](getting-started.md) — first scan, common commands
- [Configuration](configuration.md) — `.archfit.yaml` reference
- [Agent Skill](agent-skill.md) — set up Claude Code integration
