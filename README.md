# officecli-patch

`officecli-patch` 是跨平台 CLI，專門將 AI 修改後的 JSON 文字內容安全寫回原始 DOCX，同時盡量保留原文件格式。

除了完整保留原生 OfficeCLI 指令外，另外新增兩個 patch 指令：

- `diff`：比較原始 JSON 與 AI JSON，產生只含文字變更的 patch。
- `rewrite`：將原本多步驟的「複製原檔、找出文字差異、套用文字 patch」合併成一次執行。

每個 build 都只內嵌其對應平台與 CPU 架構的官方 OfficeCLI。除了 `diff`、`rewrite` 以外的命令與參數，會原封不動轉送給對應的原生 OfficeCLI。

## 使用方式

先用官方 OfficeCLI 取得原始文件的 dump JSON，並讓 AI 僅修改其中的 `props.text`，另存成 AI JSON。

./officecli-patch-win-x64.exe dump "doc/20261101午宴萬豪四季廳蔡李府.docx" -o ./output/wedding.json

```powershell
# 可選：只產生並檢查文字差異 patch
./officecli-patch-win-x64.exe diff wedding.json wedding_ai.json -o patch.json

# 一次完成：複製原檔、找出文字差異、套用文字變更
./officecli-patch-win-x64.exe rewrite "doc/20261101.docx" ./output/wedding.json ./output/wedding_ai.json -o ./output/output_ai.docx --best-effort
```

若未指定 `-o`，`rewrite` 預設輸出 `<原檔名>.rewritten.docx`。

可用旗標：

- `--force`：允許覆寫既有輸出檔。
- `--best-effort`：某個文字 run 的路徑無法套用時，仍繼續處理其他文字。官方 OfficeCLI 會顯示失敗項目，並以 exit code `2` 表示有警告。

## `rewrite` 做了什麼

原本流程可寫成：

```powershell
Copy-Item original.docx output_ai.docx
# 比較 original.json 與 ai.json，產生 patch.json
officecli batch output_ai.docx --input patch.json --best-effort
```

現在只需：

```powershell
.\officecli-patch-win-x64.exe rewrite original.docx original.json ai.json -o output_ai.docx --best-effort
```

工具刻意**不會**將完整的 `ai.json` 直接交給 `officecli batch`。完整 dump 會包含樣式、表格、圖片、header/footer、relationship 等結構內容；replay 它不符合「只改文字、保留原格式」的需求。

`rewrite` 僅從同一索引位置的既有文字 run 讀取 `props.text` 差異，產生 `set` 指令。套用後，只有實際成功更新文字的 XML part 取自 OfficeCLI 結果；其餘 DOCX ZIP entry 直接從原始文件帶回，因此可保留水印、logo、header/footer、shape、image、style、table 與 relationship。

## 原生 OfficeCLI 指令

所有非 `diff`、`rewrite` 的指令皆可直接使用，語法與官方 OfficeCLI 一致：

```powershell
# 建立 AI 可編輯的原始 JSON
.\officecli-patch-win-x64.exe dump original.docx -o original.json

# 原生檢查、查詢與 raw XML 操作
.\officecli-patch-win-x64.exe validate output_ai.docx --json
.\officecli-patch-win-x64.exe get output_ai.docx /body
.\officecli-patch-win-x64.exe raw output_ai.docx /document
.\officecli-patch-win-x64.exe help docx
```

裸用 `officecli-patch help` 顯示 wrapper 的說明；帶有參數的 native help，例如 `officecli-patch help docx`，會轉送到官方 OfficeCLI。

## 建置

```powershell
powershell.exe -ExecutionPolicy Bypass -File ./build.ps1
.\build.ps1
```

Git Bash 可直接執行：

```bash
./build.sh
```

兩個腳本預設會先從 [官方 OfficeCLI Releases](https://github.com/iOfficeAI/OfficeCLI/releases) 下載最新八種官方 binary，並以 GitHub API 提供的 SHA-256 digest 驗證後才建置。若要使用已下載的檔案而不更新，可用：

```powershell
.\build.ps1 -SkipFetch
```

```bash
OFFICECLI_PATCH_SKIP_FETCH=1 ./build.sh
```

官方原生檔案必須置於 `tools/`；所有產物都會寫入 `release/`。目前未放入的官方檔案會顯示警告並略過，放入後下次重跑 `build.ps1` 會自動加入對應產物。

| `tools/` 官方檔案 | `release/` 產物 |
| --- | --- |
| `officecli-win-x64.exe` | `officecli-patch-win-x64.exe` |
| `officecli-win-arm64.exe` | `officecli-patch-win-arm64.exe` |
| `officecli-linux-x64` | `officecli-patch-linux-x64` |
| `officecli-linux-arm64` | `officecli-patch-linux-arm64` |
| `officecli-linux-alpine-x64` | `officecli-patch-linux-alpine-x64` |
| `officecli-linux-alpine-arm64` | `officecli-patch-linux-alpine-arm64` |
| `officecli-mac-x64` | `officecli-patch-mac-x64` |
| `officecli-mac-arm64` | `officecli-patch-mac-arm64` |

首次加入或更新官方檔案時，請核對 SHA-256：

| 官方檔案 | SHA-256 |
| --- | --- |
| `officecli-linux-alpine-arm64` | `65c65e05100bac1376e23f6ca97086a046afdfcaf50f2bc215a8c139b3bc22ba` |
| `officecli-linux-alpine-x64` | `390e246303bf43b4739e3195e9b171a660223b4c11218b65894d5fadf9a52755` |
| `officecli-linux-arm64` | `bc06deaa0ad931f5208717a40b94018dc44cdff0d8eefa842c4f4daf89fb35a8` |
| `officecli-linux-x64` | `e54d3c1d248372365f0634aac56d6f1918bd04d6e71afc792ad50e075f56cfe9` |
| `officecli-mac-arm64` | `e2ed6eba5cd46d6800139f2835097828b8ccd7c8c9b679463b50e45ba2f1dbf5` |
| `officecli-mac-x64` | `5071abef56c1d4a4d60e28ed12bc66183d8dc6a9783529c3f1a9cf6bdfe6c2dd` |
| `officecli-win-arm64.exe` | `82408eca64c0f7a79679754162320aae510b26f4cf93933c9304dbd07a379c83` |
| `officecli-win-x64.exe` | `047705402974c3690a4437e55f620d03afac4beba4fdd28fdb59af610a3afff2` |

日後只需以新版官方檔案覆寫 `tools/` 中的同名檔案，再執行 `build.ps1`；產物會自動重新嵌入新版 OfficeCLI。

若受管理的環境無法寫入使用者快取，可設定 `OFFICECLI_PATCH_CACHE` 到可寫入的資料夾。

## GitHub Release 自動化

官方 binary 與本機 `release/` 產物均被 `.gitignore` 排除，不會提交到本專案。GitHub Actions [release workflow](.github/workflows/release.yml) 可從 Actions 頁面手動執行：填入版本號與官方 OfficeCLI tag（或 `latest`）後，它會下載、驗證、建置八個平台、建立 `checksums.txt`，並將所有產物上傳到**你的 GitHub Release**。

## PyPI launcher

`pypi/` 是不含 native binary 的輕量 Python 套件。使用者安裝正式發布版後：

```bash
pip install officecli-patch
officecli-patch rewrite original.docx original.json ai.json -o output.docx
```

它會自動偵測 Windows/macOS/Linux、x64/ARM64，以及 Linux 的 glibc 或 musl/Alpine；首次執行時由你的 GitHub Release 下載正確的 `officecli-patch-*` 檔案、以 Release 的 `checksums.txt` 做 SHA-256 驗證，快取後再執行。

在 GitHub Actions 的手動 Release 表單勾選 `publish_pypi`，並於 GitHub repository secrets 設定 `PYPI_API_TOKEN`，workflow 便會在建立 GitHub Release 後發布 PyPI launcher。PyPI wheel 建置時會自動嵌入目前 repository 名稱，讓 launcher 知道應從哪個 GitHub Release 下載。
