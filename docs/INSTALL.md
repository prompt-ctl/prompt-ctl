# Installation Guide

Detailed platform-specific instructions for installing promptctl.

## Quick Install

Choose the fastest option for your platform:

| Platform | Command |
|----------|---------|
| **macOS** | `brew tap oleg-koval/tap && brew install promptctl` |
| **Linux (Snap)** | `snap install promptctl` |
| **Windows** | Download binary from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases) |
| **Any (Go)** | `go install github.com/oleg-koval/promptctl@latest` |

After installation, verify with:

```bash
promptctl version
```

---

## Linux

### Ubuntu / Debian (APT)

Download the `.deb` package from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases):

```bash
# Download the latest .deb package (replace VERSION and ARCH as needed)
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_linux_amd64.deb

# Install
sudo dpkg -i promptctl_linux_amd64.deb

# Verify
promptctl version
```

### Fedora / RHEL / CentOS (RPM)

Download the `.rpm` package from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases):

```bash
# Download the latest .rpm package (replace VERSION and ARCH as needed)
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_linux_amd64.rpm

# Install
sudo rpm -i promptctl_linux_amd64.rpm

# Verify
promptctl version
```

### Arch Linux

Available via AUR:

```bash
# Using yay
yay -S promptctl

# Or using paru
paru -S promptctl
```

### Alpine Linux

Download the static binary (works on musl-based systems):

```bash
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_linux_amd64.tar.gz
tar xzf promptctl_linux_amd64.tar.gz
sudo mv promptctl /usr/local/bin/
promptctl version
```

### Snap

```bash
snap install promptctl
```

### Direct Binary

Download the tarball from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases):

```bash
# Download (replace ARCH with amd64 or arm64)
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_linux_amd64.tar.gz

# Extract
tar xzf promptctl_linux_amd64.tar.gz

# Move to PATH
sudo mv promptctl /usr/local/bin/

# Verify
promptctl version
```

---

## macOS

### Homebrew (Recommended)

```bash
brew tap oleg-koval/tap
brew install promptctl
```

To update later:

```bash
brew upgrade promptctl
```

### Direct Binary

Download from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases):

```bash
# For Apple Silicon (M1/M2/M3/M4)
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_darwin_arm64.tar.gz

# For Intel Macs
curl -LO https://github.com/oleg-koval/promptctl/releases/latest/download/promptctl_darwin_amd64.tar.gz

# Extract and install
tar xzf promptctl_darwin_*.tar.gz
sudo mv promptctl /usr/local/bin/

# Verify
promptctl version
```

> **Apple Silicon note:** promptctl is built natively for both Intel (amd64) and Apple Silicon (arm64). No Rosetta required.

---

## Windows

### Direct Binary

1. Download the latest `.zip` from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases) (e.g., `promptctl_windows_amd64.zip`).
2. Extract the archive.
3. Move `promptctl.exe` to a directory in your PATH.

**Adding to PATH:**

1. Open **Settings** > **System** > **About** > **Advanced system settings**.
2. Click **Environment Variables**.
3. Under **User variables**, select `Path` and click **Edit**.
4. Click **New** and add the directory containing `promptctl.exe` (e.g., `C:\Users\YourName\bin`).
5. Click **OK** to save.

Verify in a new terminal:

```powershell
promptctl version
```

### Chocolatey

If available:

```powershell
choco install promptctl
```

### WSL (Windows Subsystem for Linux)

If you use WSL, follow the [Linux instructions](#linux) above inside your WSL terminal.

---

## From Source

### Requirements

- [Go 1.22+](https://go.dev/dl/)
- [Git](https://git-scm.com/)

### Build and Install

```bash
git clone https://github.com/oleg-koval/promptctl.git
cd promptctl
go build -o promptctl .
```

Move the binary to your PATH:

```bash
# Linux / macOS
sudo mv promptctl /usr/local/bin/

# Or install to your Go bin directory
go install github.com/oleg-koval/promptctl@latest
```

Verify:

```bash
promptctl version
```

### Go Install (One-liner)

If you have Go installed, this is the simplest method:

```bash
go install github.com/oleg-koval/promptctl@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your PATH.

---

## Uninstall

### macOS (Homebrew)

```bash
brew uninstall promptctl
brew untap oleg-koval/tap  # optional: remove the tap
```

### macOS / Linux (Direct Binary)

```bash
sudo rm /usr/local/bin/promptctl
```

### Linux (Snap)

```bash
snap remove promptctl
```

### Linux (APT)

```bash
sudo apt remove promptctl
```

### Linux (RPM)

```bash
sudo rpm -e promptctl
```

### Windows

Delete `promptctl.exe` from its installation directory, or use **Add or Remove Programs** if installed via Chocolatey.

### Go Install

```bash
rm $(which promptctl)
```

---

## Troubleshooting

**"command not found" after install:**
Make sure the binary location is in your `PATH`. For Go installs, add `export PATH="$HOME/go/bin:$PATH"` to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.).

**Permission denied on macOS:**
If macOS blocks the binary, go to **System Settings** > **Privacy & Security** and click "Allow Anyway".

**Need help?** Open an issue at [github.com/oleg-koval/promptctl/issues](https://github.com/oleg-koval/promptctl/issues).
