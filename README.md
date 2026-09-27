# officecli-patch

```bash
pip install officecli-patch
```

安裝後可直接使用：

```bash
officecli-patch rewrite original.docx original.json ai.json -o output.docx
```

`officecli-patch` 基於 [iOfficeAI OfficeCLI](https://github.com/iOfficeAI/OfficeCLI)。官方 OfficeCLI 提供完整的 Office 文件操作能力，但目前沒有「以原始 DOCX 為基礎，只套用 AI JSON 的文字變更並盡量保留既有格式與內容」的專用流程。

本倉儲補足此功能：`rewrite` 只讀取 `props.text` 差異，將文字套回原始文件，並保留未修改的 DOCX package part，避免重建整份文件。它適合需要修改文字、但希望保留原本 watermark、logo、header/footer、圖片、shape、樣式、表格與 relationship 的情境。

除了新增的 `diff`、`rewrite`，**所有原生 OfficeCLI 指令與參數都會原封不動 passthrough**。

## 使用方式

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

## 自訂指令

| 指令 | 說明 |
| --- | --- |
| `diff <original.json> <ai.json> [-o <patch.json>]` | 僅產生既有文字 run 的 `props.text` 差異。 |
| `rewrite <source.docx> <original.json> <ai.json> [-o <output.docx>]` | 複製原文件、產生文字 patch、套用變更並還原未修改部分。 |

`rewrite` 預設輸出 `<原檔名>.rewritten.docx`。使用 `--best-effort` 時，無法套用的個別 run 不會阻斷其餘文字更新；使用 `--force` 可覆寫既有輸出檔。

## PyPI 與 GitHub Release

PyPI 套件是 launcher：首次執行時會自動判斷 Windows/macOS/Linux、x64/ARM64、Linux glibc/musl，從本專案的 GitHub Release 下載正確 binary、驗證 `checksums.txt` 後執行。

維護者建置與發佈：

```bash
./build.sh
```

或在 Windows：

```powershell
.\build.ps1
```

兩個腳本會自動從 [OfficeCLI 官方 Releases](https://github.com/iOfficeAI/OfficeCLI/releases) 下載並驗證官方 binaries，再將各平台輸出放入 `release/`。`tools/` 的官方 binary 與本機 `release/` 產物均不會提交至 Git。

GitHub Actions 的 **Build and release officecli-patch** workflow 可手動建立 GitHub Release；勾選 `publish_pypi` 可透過 PyPI Trusted Publishing 發布 launcher。首次發布前，請在 PyPI 專案設定 GitHub Trusted Publisher：repository 為你的 `owner/officecli-patch`、workflow 為 `release.yml`。
