# SpendSense CLI Installation Guide

SpendSense CLI is a command-line tool for managing your personal expenses directly from your terminal.

## Download

Download the appropriate binary for your operating system from the GitHub Releases page.

| Platform              | Architecture | File                                       |
| --------------------- | ------------ | ------------------------------------------ |
| macOS (Intel)         | amd64        | `spendsense-cli_0.1.1_darwin_amd64.tar.gz` |
| macOS (Apple Silicon) | arm64        | `spendsense-cli_0.1.1_darwin_arm64.tar.gz` |
| Linux (x86_64)        | amd64        | `spendsense-cli_0.1.1_linux_amd64.tar.gz`  |
| Linux (ARM64)         | arm64        | `spendsense-cli_0.1.1_linux_arm64.tar.gz`  |
| Windows (64-bit)      | amd64        | `spendsense-cli_0.1.1_windows_amd64.zip`   |

---

## Linux Installation

### 1. Extract the archive

```bash
tar -xzf spendsense-cli_0.1.1_linux_amd64.tar.gz
```

### 2. Make the binary executable

```bash
chmod +x spendsense
```

### 3. Install system-wide (optional)

```bash
sudo mv spendsense /usr/local/bin/
```

### 4. Verify installation

```bash
spendsense --help
```

---

## macOS Installation

### Intel Macs

```bash
tar -xzf spendsense-cli_0.1.1_darwin_amd64.tar.gz
chmod +x spendsense
sudo mv spendsense /usr/local/bin/
```

### Apple Silicon (M1/M2/M3/M4)

```bash
tar -xzf spendsense-cli_0.1.1_darwin_arm64.tar.gz
chmod +x spendsense
sudo mv spendsense /usr/local/bin/
```

### Verify installation

```bash
spendsense --help
```

If macOS blocks the binary because it is unsigned:

```bash
xattr -d com.apple.quarantine spendsense
```

---

## Windows Installation

### 1. Extract the ZIP archive

Extract:

```text
spendsense-cli_0.1.1_windows_amd64.zip
```

### 2. Move the executable

Move `spendsense.exe` to a directory of your choice, for example:

```text
C:\Tools\SpendSense\
```

### 3. Add the directory to PATH

Add the installation directory to your system PATH.

### 4. Verify installation

Open PowerShell or Command Prompt:

```powershell
spendsense --help
```

---

## Verify File Integrity (Optional)

You can verify the downloaded file using SHA256 checksums.

### Linux/macOS

```bash
sha256sum spendsense-cli_0.1.1_linux_amd64.tar.gz
```

Expected output:

```text
62e9c333d8c80ab6b69fc0c0b003d3c03088a92172aefee694ca8e39da61d82e
```

### macOS Intel

```text
72d030bcf228af655b7b92649356444a8eaeaf32043751b0ab079d34ae557353
```

### macOS Apple Silicon

```text
405321964375d1d9c62ce561dba89e167a217dff3dc844098bf43212a8075131
```

### Linux x86_64

```text
62e9c333d8c80ab6b69fc0c0b003d3c03088a92172aefee694ca8e39da61d82e
```

### Linux ARM64

```text
4d0db4dfea28f83c1c2b6a1dbec4df29458ee13ae087a6e6412fc03a4ef384d1
```

### Windows x86_64

```text
0620d960ef8d44e279b6e635ca100e1407f1677d918d4c220d71a41fccd4beb4
```

---

## Getting Started

Register a new account:

```bash
spendsense register --email user@example.com
```

Login:

```bash
spendsense login --email user@example.com
```

Add an expense:

```bash
spendsense add \
  --amount 50 \
  --category Food \
  --date today \
  --merchant "Cafe"
```

List expenses:

```bash
spendsense list
```

Logout:

```bash
spendsense logout
```

---

## Configuration

SpendSense stores local CLI configuration and authentication information in:

```text
~/.expenserc
```

Do not share this file with others.

---

## Support

If you encounter issues, please open an issue in the GitHub repository with:

* Operating system
* SpendSense CLI version
* Command executed
* Error output
