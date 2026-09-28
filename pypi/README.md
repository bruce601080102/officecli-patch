# officecli-patch

[繁體中文](#繁體中文) · [English](#english)

## 繁體中文

`officecli-patch` 是 [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI) 的相容擴充工具。它保留官方 OfficeCLI 的所有原生指令與參數，並新增 `diff` 與 `rewrite`。

### 為什麼會有這個套件？

官方 OfficeCLI 很適合 dump、檢查與批次處理 Office 文件。不過，當 DOCX、XLSX 或 PPTX 已有既定格式時，直接回寫 AI 產生的完整 JSON，可能影響原本不該變動的文件內容。

常見需求其實是：**只修改 AI 真正改過的內容，同時保留原始 Office 文件的圖片、shape、表格、樣式與 relationship。**

官方 OfficeCLI 目前尚未提供這個「以原始文件為基礎、執行完整 AI JSON batch 後再以內容 patch 修正」的整合流程。`officecli-patch` 因此將手動複製文件、執行 AI JSON batch、比較 JSON、產生 patch、再執行 batch 的流程，收斂為 `rewrite` 指令。內容 patch 支援 DOCX 文字 run、PPTX 文字，以及 XLSX 工作表 import 與 rich text。

XLSX 的 `import.text` 差異會被拆成個別儲存格指令，只更新真正變動的儲存格，避免未修改的 rich text 或格式流失。

### 安裝與使用

```bash
python -m pip install --upgrade officecli-patch
officecli-patch --version

# 先 dump 原始文件；AI 修改內容欄位後另存 ai.json
officecli-patch dump original.xlsx -o original.json
officecli-patch rewrite original.xlsx original.json ai.json -o rewritten.xlsx
```

pip 會安裝 `officecli-patch` 終端命令。第一次執行時會下載並驗證符合目前平台的原生執行檔。如果終端顯示找不到命令，請重新開啟終端或將 Python 的 Scripts/bin 目錄加入 `PATH`；也可用 `python -m officecli_patch --version` 作為不依賴 `PATH` 的備援入口。

若只需要查看內容差異：

```bash
officecli-patch diff original.json ai.json -o patch.json
```

其他命令都會直接轉交官方 OfficeCLI：

```bash
officecli-patch validate original.docx
officecli-patch batch document.docx --input commands.json
officecli-patch raw --help
```

首次執行時，PyPI 啟動器會自動判斷作業系統、CPU 架構與 Linux libc 類型，從與該 PyPI 版本相同的 GitHub Release 下載對應執行檔、驗證 SHA-256 並快取到本機。每個版本使用獨立快取，不會重用舊版 binary。

## English

`officecli-patch` is a compatible extension for [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI). It preserves all official OfficeCLI commands and arguments, and adds `diff` and `rewrite`.

### Why does this package exist?

OfficeCLI is excellent for dumping, inspecting, and batch-processing Office documents. But for a document with an established design—such as letterhead, a contract, a wedding menu, or a company template—writing back an entire AI-generated JSON document can alter content that was never meant to change.

The practical requirement is usually: **change only the content the AI actually changed, while retaining the original Office document's images, shapes, tables, styles, and relationships.**

Official OfficeCLI does not yet provide an integrated command for the workflow that starts from the original document, runs the complete AI JSON batch, and then corrects it with a content patch. `officecli-patch` provides that workflow through `rewrite`, replacing the manual copy → AI JSON batch → JSON comparison → patch generation → best-effort batch sequence. Content patches support DOCX runs, PPTX text, and XLSX worksheet imports/rich text.

XLSX `import.text` differences are expanded into per-cell commands so unchanged rich text and formatting are retained.

### Install and use

```bash
python -m pip install --upgrade officecli-patch
officecli-patch --version

# Dump the source document. Let the AI edit content fields in original.json,
# then save it as ai.json.
officecli-patch dump original.pptx -o original.json
officecli-patch rewrite original.pptx original.json ai.json -o rewritten.pptx
```

pip installs an `officecli-patch` terminal command. On first use it downloads and verifies the native executable for the current platform. If the command is not found, reopen the terminal or add Python's Scripts/bin directory to `PATH`; `python -m officecli_patch --version` is a PATH-independent fallback.

To inspect or save the content patch only:

```bash
officecli-patch diff original.json ai.json -o patch.json
```

Every other command is passed through to the embedded official OfficeCLI:

```bash
officecli-patch validate original.docx
officecli-patch batch document.docx --input commands.json
officecli-patch raw --help
```

On first use, the PyPI launcher detects the operating system, CPU architecture, and Linux libc variant. It downloads the matching binary from the GitHub Release for that exact PyPI version, verifies its SHA-256 checksum, then caches and runs it. Each release has its own cache, so an older binary is never reused after an upgrade.

Source code, full documentation, and releases: [bruce601080102/officecli-patch](https://github.com/bruce601080102/officecli-patch).
