---
task_id: "TASK-0064"
status: complete
completed_at: "2026-08-02T02:20:38Z"
candidate_commit: "669ac3f95dd929f9e5a12ad019c5998df9d1fe6f"
---

# TASK-0064 HANDOVER

## 成果

- `make lint-docs`の既存3検査をshellなしの小さなNode runnerへ配線し、先行FAIL又はspawn failure後も固定順で全検査を一回ずつ実行するようにした。
- 全PASSだけzero、一件以上のFAILは全検査終了後にnon-zeroとし、子processのstdioを継承して同じ起動内で全指摘を表示する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate固定直前） | `PASS` |
| `node --test scripts/task/development-process.test.mjs` | `PASS`（72/72） |
| `make lint-docs` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み3パス、138 additions / 3 deletions。既存`test:process`列挙済みtestへfake spawn casesを追加し、新runnerのtestがroot checkから常時実行される。
- terminology、textlint、diff checkの内容・順序、lint rule、用語集、対象Markdown、`make check`の他targetは変更していない。

## 検証結果

- `make check`: `PASS`
- `node --test scripts/task/development-process.test.mjs`: `PASS`（72/72）
- `make lint-docs`: `PASS`
- `git diff --check`: `PASS`

## 判断・既知の制約

- 初回`make lint-docs`はsandbox DNS制約による依存導入失敗で、実装不具合ではなくenvironmentに分類した。許可済み環境で再実行し、3検査全てと最終exit 0を確認した。
- Main差分確認でspawn error診断の可変command/error露出をcandidate前に除き、固定文だけをreportするよう修正した。returned errorとthrowの双方をunit testで確認した。
- network、OS権限、credential、live配置を扱わないためlive E2Eはない。
