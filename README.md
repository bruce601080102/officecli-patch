# officecli-patch

[繁體中文](#繁體中文) · [English](#english)

## 繁體中文

```bash
python -m pip install --upgrade officecli-patch
officecli-patch --version
```

安裝後，pip 會建立 `officecli-patch` 終端命令，可直接使用：

```bash
officecli-patch rewrite original.xlsx original.json ai.json -o output.xlsx
```

第一次執行會下載並驗證與套件版本相同、符合目前作業系統與 CPU 的原生執行檔。如果終端顯示找不到命令，代表 Python 的 Scripts/bin 目錄不在 `PATH`；可重新開啟終端，或先用 `python -m officecli_patch --version` 執行相同入口。在 virtualenv／venv 中安裝時，啟用環境後即可直接使用 `officecli-patch`。

`officecli-patch` 基於 [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI)。官方 OfficeCLI 提供完整的 Office 文件操作能力，但目前沒有「以原始 Office 文件為基礎，只套用 AI JSON 的內容變更並盡量保留既有格式與內容」的專用流程。

本專案補足此功能：`rewrite` 整合既有的官方 OfficeCLI 工作流程：先複製原文件、執行完整 AI JSON batch、產生窄範圍內容 patch，最後以 `--best-effort` 套用 patch。即使前一個 batch 因個別路徑錯誤而 atomic rollback，仍會繼續執行最後的 patch。內容 patch 支援 DOCX 文字 run、PPTX 文字，以及 XLSX 工作表 import 與 rich text；不會把 AI 意外改動的樣式帶進 fallback patch。

XLSX 的 `import.text` 差異會被拆成個別儲存格的 `set`／`clear`／`formula` 指令，只更新真正變動的儲存格，避免整張重新 import 導致未修改的 rich text 或格式流失。

除了新增的 `diff`、`rewrite`，所有原生 OfficeCLI 指令與參數都會直接 passthrough。

### 使用方式

先以原生 `dump` 取得 JSON，再讓 AI 修改內容欄位。DOCX/PPTX 通常是 `props.text`，XLSX 的一般儲存格內容位於 `import.text`，rich text 位於 `props.runs`：

```bash
officecli-patch dump original.xlsx -o original.json
# 將 original.json 交給 AI 編輯，另存為 ai.json

# 可選：檢視內容差異 patch
officecli-patch diff original.json ai.json -o patch.json

# 建立保留原格式的輸出文件
officecli-patch rewrite original.xlsx original.json ai.json -o output.xlsx
```

原生指令照常可用：

```bash
officecli-patch validate output.docx --json
officecli-patch get output.docx /body
officecli-patch raw output.docx /document
officecli-patch help docx
```

| 指令 | 說明 |
| --- | --- |
| `diff <original.json> <ai.json> [-o <patch.json>]` | 產生 DOCX、PPTX、XLSX 的內容差異 patch，不包含樣式差異。 |
| `rewrite <source.docx\|xlsx\|pptx> <original.json> <ai.json> [-o <output>]` | 複製原文件、執行完整 AI JSON batch、產生內容 patch，再以 `--best-effort` 套用 patch。 |

`rewrite` 預設輸出 `<原檔名>.rewritten.<原副檔名>`，並固定以 `--best-effort` 套用 patch：無法套用的個別項目不會取消其他成功的內容更新。`--force` 可覆寫既有輸出檔。

### PyPI 與 GitHub Release

PyPI 套件是 launcher：首次執行時會自動判斷 Windows/macOS/Linux、x64/ARM64、Linux glibc/musl，從與 PyPI 套件相同版本的 GitHub Release 下載正確 binary、驗證 `checksums.txt` 後執行。每個 Release 使用獨立快取，不會重用舊 binary。

維護者可執行 `./build.sh`，或在 Windows 執行 `./build.ps1`。兩者都會下載、驗證官方 OfficeCLI binaries，並將各平台輸出放入 `release/`；官方 binary 與本機產物不會提交至 Git。

## English

```bash
python -m pip install --upgrade officecli-patch
officecli-patch --version
```

pip installs an `officecli-patch` terminal command. The first run downloads and verifies the native executable matching the package version, operating system, and CPU. If the command is not found, reopen the terminal or add Python's Scripts/bin directory to `PATH`; `python -m officecli_patch --version` is a PATH-independent fallback. An activated virtual environment exposes `officecli-patch` directly.

`officecli-patch` is a compatible extension for [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI). It keeps every native OfficeCLI command and argument, while adding `diff` and `rewrite`.

OfficeCLI does not currently provide a dedicated command for this original-format rewrite workflow. `rewrite` copies the source DOCX, XLSX, or PPTX, runs the complete AI JSON through batch, generates a narrow content patch, then batch-applies that patch with `--best-effort`. The fallback patch supports DOCX runs, PPTX text, and XLSX worksheet imports/rich text.

XLSX `import.text` differences are expanded into per-cell `set`, `clear`, or `formula` commands. Only changed cells are touched, so unchanged rich text and formatting are retained.

This is useful for documents that must retain their watermark, logo, headers/footers, images, shapes, styles, tables, and relationships.

### Usage

```bash
# Dump with the native OfficeCLI-compatible command.
officecli-patch dump original.pptx -o original.json

# Let an AI edit content fields and save the result as ai.json.
officecli-patch diff original.json ai.json -o patch.json
officecli-patch rewrite original.pptx original.json ai.json -o output.pptx
```

All other commands are passed through to the embedded official OfficeCLI:

```bash
officecli-patch validate output.docx --json
officecli-patch batch document.docx --input commands.json
officecli-patch raw output.docx /document
```

| Command | Description |
| --- | --- |
| `diff <original.json> <ai.json> [-o <patch.json>]` | Produces content-only DOCX, XLSX, and PPTX changes while excluding style changes. |
| `rewrite <source.docx\|xlsx\|pptx> <original.json> <ai.json> [-o <output>]` | Copies the source, runs the complete AI JSON batch, then creates and applies a `--best-effort` content patch. |

The PyPI package is a launcher. On first run it detects the OS, CPU architecture, and Linux libc variant; it downloads the matching binary from the GitHub Release for that PyPI version, verifies `checksums.txt`, then caches and runs it. Each release has a separate cache.

For maintainers, `./build.sh` or `./build.ps1` downloads and verifies official OfficeCLI binaries, then writes all platform builds to `release/`.
