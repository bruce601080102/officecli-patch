# officecli-patch

[繁體中文](#繁體中文) · [English](#english)

## 繁體中文

`officecli-patch` 是 [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI) 的相容擴充工具。它保留官方 OfficeCLI 的所有原生指令與參數，並新增 `diff` 與 `rewrite`。

### 為什麼會有這個套件？

官方 OfficeCLI 很適合 dump、檢查與批次處理 Office 文件。不過，當文件已有既定格式，例如信紙、合約、婚禮菜單或公司範本時，直接回寫 AI 產生的完整 JSON，可能影響原本不該變動的文件內容。

常見需求其實是：**只修改 AI 真正改過的文字，同時保留原始 DOCX 的 watermark、logo、頁首／頁尾、圖片、shape、表格、樣式與 relationship。**

官方 OfficeCLI 目前尚未提供這個「以原始文件為基礎、執行完整 AI JSON batch 後再以文字 patch 修正格式」的整合流程。`officecli-patch` 因此將手動複製文件、執行 AI JSON batch、比較 JSON、產生 patch、再執行 batch 的流程，收斂為 `rewrite` 指令。

### 安裝與使用

```bash
pip install officecli-patch

# 先 dump 原始文件；AI 只修改 original.json 中的 props.text 後另存 ai.json
officecli-patch dump original.docx -o original.json
officecli-patch rewrite original.docx original.json ai.json -o rewritten.docx
```

若只需要查看文字差異：

```bash
officecli-patch diff original.json ai.json -o patch.json
```

其他命令都會直接轉交官方 OfficeCLI：

```bash
officecli-patch validate original.docx
officecli-patch batch document.docx --input commands.json
officecli-patch raw --help
```

首次執行時，PyPI 啟動器會自動判斷作業系統、CPU 架構與 Linux libc 類型，從 GitHub Release 下載對應執行檔、驗證 SHA-256 並快取到本機。

## English

`officecli-patch` is a compatible extension for [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI). It preserves all official OfficeCLI commands and arguments, and adds `diff` and `rewrite`.

### Why does this package exist?

OfficeCLI is excellent for dumping, inspecting, and batch-processing Office documents. But for a document with an established design—such as letterhead, a contract, a wedding menu, or a company template—writing back an entire AI-generated JSON document can alter content that was never meant to change.

The practical requirement is usually: **change only the text the AI actually changed, while retaining the original DOCX watermark, logo, headers/footers, images, shapes, tables, styles, and relationships.**

Official OfficeCLI does not yet provide an integrated command for the workflow that starts from the original document, runs the complete AI JSON batch, and then corrects it with a text patch. `officecli-patch` provides that workflow through `rewrite`, replacing the manual copy → AI JSON batch → JSON comparison → patch generation → best-effort batch sequence.

### Install and use

```bash
pip install officecli-patch

# Dump the source document. Let the AI edit only props.text in original.json,
# then save it as ai.json.
officecli-patch dump original.docx -o original.json
officecli-patch rewrite original.docx original.json ai.json -o rewritten.docx
```

To inspect or save the text patch only:

```bash
officecli-patch diff original.json ai.json -o patch.json
```

Every other command is passed through to the embedded official OfficeCLI:

```bash
officecli-patch validate original.docx
officecli-patch batch document.docx --input commands.json
officecli-patch raw --help
```

On first use, the PyPI launcher detects the operating system, CPU architecture, and Linux libc variant. It downloads the matching GitHub Release binary, verifies its SHA-256 checksum, caches it locally, and runs it.

Source code, full documentation, and releases: [bruce601080102/officecli-patch](https://github.com/bruce601080102/officecli-patch).
