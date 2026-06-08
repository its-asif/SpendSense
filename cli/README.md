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


## Getting Started

Register a new account:

```bash
spendsense auth register
```

Login:

```bash
spendsense auth login
```

Add an expense:

```bash
spendsense expense add \
  --amount 50 \
  --category Food \
  --date today \
  --merchant "Cafe"
```

List expenses:

```bash
spendsense expense list
```

Delete an expense:

```bash
spendsense expense delete <expense-id>
```

Add an income:

```bash
spendsense income add \
  --amount 1500 \
  --wallet "Cash Wallet" \
  --category "Salary" \
  --source "Salary Paycheck"
```

List incomes:

```bash
spendsense income list
```

Delete an income:

```bash
spendsense income delete <income-id>
```

View default categories:

```bash
spendsense category list
```

View active wallets:

```bash
spendsense wallet list
```

Configuration commands:

```bash
# View configuration
spendsense config view

# Set a configuration value
spendsense config set api_url http://localhost:8080

# Get a configuration value
spendsense config get api_url
```

Logout:

```bash
spendsense auth logout
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
