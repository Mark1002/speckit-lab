# speckit-lab

用 [GitHub Spec Kit](https://github.com/github/spec-kit) 練習規格驅動開發（Spec-Driven Development）的專案。題目是官方 [Quickstart](https://github.github.io/spec-kit/quickstart.html) 的 Taskify，AI 代理從 Copilot 換成 Claude Code。

## Taskify 是什麼

一個團隊用的看板（Kanban）網頁程式，第一版的範圍：

- 5 個預設使用者：1 個產品經理、4 個工程師，沒有登入，從清單選自己。
- 3 個範例專案，也可以新增專案。
- 任務放在 4 個欄位：To Do、In Progress、In Review、Done，可以拖拉或用下拉選單移動。
- 任務可以指派人、修改；可以留言，只有作者能改或刪自己的留言。

## Spec Kit 流程

照官方 Quickstart 的順序走完一輪：

| 步驟 | 指令 | 產出 |
| --- | --- | --- |
| 1 | `/speckit-constitution` | [`constitution.md`](.specify/memory/constitution.md)：5 條專案原則 |
| 2 | `/speckit-specify` | [`spec.md`](specs/001-taskify-kanban/spec.md)：要做什麼、不做什麼 |
| 3 | `/speckit-clarify` | `spec.md` 的 Clarifications：5 個問題的答案 |
| 4 | `/speckit-plan` | [`plan.md`](specs/001-taskify-kanban/plan.md)、[`research.md`](specs/001-taskify-kanban/research.md)、[`data-model.md`](specs/001-taskify-kanban/data-model.md)、[`contracts/api.md`](specs/001-taskify-kanban/contracts/api.md)、[`quickstart.md`](specs/001-taskify-kanban/quickstart.md) |
| 5 | `/speckit-checklist` | [`checklists/security.md`](specs/001-taskify-kanban/checklists/security.md)：15 項需求品質檢查 |
| 6 | `/speckit-tasks` | [`tasks.md`](specs/001-taskify-kanban/tasks.md)：36 個任務 |
| 7 | `/speckit-analyze` | 文件一致性報告（不改檔案），修正 9 項 |
| 8 | `/speckit-implement` | 分 3 段實作：MVP、其餘使用者故事、最後檢查 |
| 9 | `/speckit-converge` | 第一次補了 4 個任務（T037–T040），第二次回報 Converged |

## 跟官方 Quickstart 不同的地方

| 項目 | 官方 | 這個專案 | 原因 |
| --- | --- | --- | --- |
| 初始化 | `specify init taskify --integration copilot` | `specify init --here --integration claude` | 改用 Claude Code，直接用現有資料夾 |
| 架構 | 微服務 | 單一程式 | 想要簡單；constitution 原則 III 禁止微服務 |
| 後端 | .NET Aspire | Go 1.25 標準函式庫，0 個第三方套件 | constitution 原則 IV：套件越少越好 |
| 資料庫 | Postgres | 一個 JSON 檔，原子寫入 | 不用安裝任何東西 |
| 前端 | Blazor Server | 純 HTML、CSS、JavaScript，沒有 build | 同上 |
| 即時更新、通知 | 有 | 沒有，重新整理就看得到 | spec 把它們列為這次不做 |
| 對外連線 | 未規定 | 只聽 `127.0.0.1` | 沒有登入，所以只讓本機連線 |

指令名稱兩邊一樣，都是 `/speckit-*`。

## 執行

需要 Go 1.25 以上。

```sh
go test ./...
go run .
```

打開 <http://127.0.0.1:8080/>。第一次啟動會產生 `taskify.json`（範例資料，不進 git），刪掉它就能重來。可用 `-port` 換 port、`-data` 換資料檔。

更多檢查步驟見 [`quickstart.md`](specs/001-taskify-kanban/quickstart.md)。

## 檔案

```text
.specify/                    Spec Kit 的模板、腳本和 constitution
specs/001-taskify-kanban/    這個功能的所有規格文件
main.go  api.go  store.go  validate.go  seed.go   後端
*_test.go                    測試
web/                         前端（編進執行檔裡）
```

`.claude/`（Spec Kit 裝給 Claude Code 的 `/speckit-*` 技能）沒有放進 repo。clone 之後要用的話，先看 `specify integration --help`。

## 已知事項

- 任務 T036（`govulncheck` 漏洞掃描）刻意保持未完成：專案停在 `go1.25.0`，掃描找到 20 個 Go 標準函式庫的漏洞，`go1.25.13` 已修好。照 constitution，這版不算正式發布。
- 漏洞掃描結果取決於執行它的 Go 版本，要用編譯發布版的同一個 Go 來掃。
