# ComplianceProbe 🛡️

[![Build Status](https://img.shields.io/github/actions/workflow/status/benedictjohannes/ComplianceProbe/release.yml?style=flat-square)](https://github.com/benedictjohannes/ComplianceProbe/actions) [![License: MIT](https://img.shields.io/github/license/benedictjohannes/ComplianceProbe?color=yellow&style=flat-square)](https://github.com/benedictjohannes/ComplianceProbe/blob/master/LICENSE)

**ComplianceProbe** is a cross-platform security compliance reporting agent. It executes a series of automated checks defined in a YAML "playbook" to verify system integrity, security configurations, and hardware state.

Whether you are auditing a desktop for security standards or monitoring server health, ComplianceProbe provides a flexible, scriptable, and reproducible way to generate detailed compliance reports.

## ✨ Key Features

-   **🔍 Automated Compliance Checks**: Group assertions into logical sections (e.g., OS Integrity, IAM, Data Protection).
-   **🚀 Multi-Platform support**: Native binaries for Linux, Windows, and macOS (Intel & ARM).
-   **JS Scripting & Logic**: Dynamic script generation and output evaluation using an embedded JavaScript engine (Goja).
-   **📊 Comprehensive Reporting**: Generates reports in three formats:
    -   **Markdown**: Human-readable summary for documentation.
    -   **JSON**: Machine-readable data for integration with other tools.
    -   **Detailed Logs**: Full execution trace for debugging.
-   **📥 Data Gathering**: Extract information from command outputs (via Regex or JS) and reuse it in subsequent checks within the same assertion.
-   **🛠️ Preprocessing Pipeline**: Write complex logic in separate `.js` or `.ts` files and "bake" them into a single portable playbook using the builder tool.
-   **✅ Schema Validation**: Built-in JSON schema generation to ensure your playbooks are correctly formatted.

## 🎯 Use Cases

-   **🌍 Adaptive Fleet Audits**: Run compliance checks across Linux, Windows, and macOS using a single **"Universal Playbook"** that adapts logic at runtime via JavaScript, or maintain **platform-specific playbooks** for targeted simplicity.
-   **🛡️ Dynamic Security Chaining**: Go beyond static checks by extracting data (like current user or PID) in one step and using it to drive subsequent commands within the same assertion.
-   **� Privacy-Aware Secret Validation**: Audit sensitive configurations for keys or PII without leaking them. Extract values for internal logic while explicitly excluding them from the final JSON/Markdown reports.
-   **📈 Weighted Compliance Scoring**: Move past binary Pass/Fail results. Assign scores to assertions to generate a numerical "Security Health" grade for your systems.
-   **🛠️ Pre-Flight Environment Checks**: Verify system integrity—from kernel versions to script syntax—before deploying applications or onboarding new developer machines.

## �📦 Installation

Download the binary for your platform from the [releases](https://github.com/benedictjohannes/ComplianceProbe/releases) page:

-   `compliance-probe-linux`
-   `compliance-probe-windows.exe`
-   `compliance-probe-mac-arm` (for Apple Silicon)
-   `compliance-probe-mac-intel`

## 🚀 Quick Start

1.  **Run with the default playbook:**
    Ensure a `playbook.yaml` exists in the current directory and run:
    ```bash
    ./compliance-probe
    ```

2.  **Run with a specific playbook:**
    ```bash
    ./compliance-probe my-security-audit.yaml
    ```

3.  **View results:**
    Reports are saved to the `reports/` directory with a timestamped filename (e.g., `260206-033831.report.md`).

## 🛠️ Configuration (playbook.yaml)

The playbook is the heart of ComplianceProbe. It defines what to check, how to score results, and how to extract data.

For a comprehensive guide on all available features—including **weighted scoring**, **embedded JavaScript logic**, **data gathering**, and **cross-platform handling**—see the:

👉 **[playbook.example.yaml](./playbook.example.yaml)**

## 🏗️ Development and Building

The project uses Go's build tags to separate the runtime agent from the developer tools.

### Prerequisites

- [Go](https://go.dev/dl/) 1.24+

### Build Agent Binaries

To build the optimized agent binaries for all platforms:

```bash
make build
```

Or build manually for your current platform:
```bash
go build -o compliance-probe .
```

### Build Builder Binaries

The builder version includes schema generation and playbook preprocessing:

```bash
make build-builder
```

Or build manually:
```bash
go build --tags builder -o compliance-probe-builder .
```

### Running Tests

```bash
make test
```

## 🏗️ Developer Tools

ComplianceProbe includes a **Builder** personality (`compliance-probe-builder`) designed for developers creating and managing complex playbooks.

-   **Generate Schema**: Create a JSON schema for IDE autocompletion (VS Code, etc).
-   **Preprocessing Pipeline**: Use `funcFile` to externalize logic into TypeScript files, which are then transpiled and "baked" into the final playbook.

For a detailed guide on using **TypeScript**, external scripts, and the preprocessing pipeline, see:

👉 **[Playbook Development Guide](./PLAYBOOK_DEVELOPMENT.md)**

## ⚖️ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
