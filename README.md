# ComplianceProbe 🛡️

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

## 📦 Installation

Download the binary for your platform from the [releases](https://github.com/your-repo/ComplianceProbe/releases) page:

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

- [Go](https://go.dev/dl/) 1.24.5+

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

ComplianceProbe includes a "builder" mode for developers creating complex playbooks.

-   **Generate Schema**: Create a JSON schema for IDE autocompletion:
    ```bash
    make schema
    ```
-   **Preprocess**: Bake external JS/TS files into a standalone YAML:
    ```bash
    ./compliance-probe-builder --preprocess --input raw-playbook.yaml --output playbook.yaml
    ```

## ⚖️ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
