# officecli-patch

[繁體中文](#繁體中文) · [English](#english)

## 繁體中文

```bash
pip install officecli-patch
```

安裝後可直接使用：

```bash
officecli-patch rewrite original.docx original.json ai.json -o output.docx
```

`officecli-patch` 基於 [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI)。官方 OfficeCLI 提供完整的 Office 文件操作能力，但目前沒有「以原始 DOCX 為基礎，只套用 AI JSON 的文字變更並盡量保留既有格式與內容」的專用流程。

本專案補足此功能：`rewrite` 只讀取 `props.text` 差異，將文字套回原始文件，並保留未修改的 DOCX package part，避免重建整份文件。它適合需要修改文字、但希望保留 watermark、logo、頁首／頁尾、圖片、shape、樣式、表格與 relationship 的情境。

除了新增的 `diff`、`rewrite`，所有原生 OfficeCLI 指令與參數都會直接 passthrough。

### 使用方式

先以原生 `dump` 取得 JSON，再讓 AI 只修改其中的 `props.text`：

```bash
officecli-patch dump original.docx -o original.json
# 將 original.json 交給 AI 編輯，另存為 ai.json

# 可選：檢視文字差異 patch
officecli-patch diff original.json ai.json -o patch.json

# 建立保留原格式的輸出文件
officecli-patch rewrite original.docx original.json ai.json -o output.docx --best-effort
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
| `diff <original.json> <ai.json> [-o <patch.json>]` | 僅產生既有文字 run 的 `props.text` 差異。 |
| `rewrite <source.docx> <original.json> <ai.json> [-o <output.docx>]` | 複製原文件、產生文字 patch、套用變更並還原未修改部分。 |

`rewrite` 預設輸出 `<原檔名>.rewritten.docx`。`--best-effort` 會略過無法套用的個別 run；`--force` 可覆寫既有輸出檔。

### PyPI 與 GitHub Release

PyPI 套件是 launcher：首次執行時會自動判斷 Windows/macOS/Linux、x64/ARM64、Linux glibc/musl，從 GitHub Release 下載正確 binary、驗證 `checksums.txt` 後執行。

維護者可執行 `./build.sh`，或在 Windows 執行 `./build.ps1`。兩者都會下載、驗證官方 OfficeCLI binaries，並將各平台輸出放入 `release/`；官方 binary 與本機產物不會提交至 Git。

## English

```bash
pip install officecli-patch
```

`officecli-patch` is a compatible extension for [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI). It keeps every native OfficeCLI command and argument, while adding `diff` and `rewrite`.

OfficeCLI does not currently provide a dedicated workflow that starts from the original DOCX, applies only AI JSON text changes, and preserves the rest of the document. `rewrite` fills that gap: it compares only `props.text`, applies the changed text to a copy of the original document, and retains untouched DOCX package parts instead of rebuilding the entire file.

This is useful for documents that must retain their watermark, logo, headers/footers, images, shapes, styles, tables, and relationships.

### Usage

```bash
# Dump with the native OfficeCLI-compatible command.
officecli-patch dump original.docx -o original.json

# Let an AI edit only props.text and save the result as ai.json.
officecli-patch diff original.json ai.json -o patch.json
officecli-patch rewrite original.docx original.json ai.json -o output.docx --best-effort
```

All other commands are passed through to the embedded official OfficeCLI:

```bash
officecli-patch validate output.docx --json
officecli-patch batch document.docx --input commands.json
officecli-patch raw output.docx /document
```

| Command | Description |
| --- | --- |
| `diff <original.json> <ai.json> [-o <patch.json>]` | Produces `props.text` changes for existing text runs only. |
| `rewrite <source.docx> <original.json> <ai.json> [-o <output.docx>]` | Copies the source, creates a text patch, applies it, and restores untouched parts. |

The PyPI package is a launcher. On first run it detects the OS, CPU architecture, and Linux libc variant; it downloads the matching GitHub Release binary, verifies `checksums.txt`, then caches and runs it.

For maintainers, `./build.sh` or `./build.ps1` downloads and verifies official OfficeCLI binaries, then writes all platform builds to `release/`.
