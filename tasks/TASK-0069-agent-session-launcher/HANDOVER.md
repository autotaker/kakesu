---
task_id: "TASK-0069"
status: complete
completed_at: "2026-08-02T05:48:26Z"
candidate_commit: "7157a1358ab2789cf72b22f973734fa833d496ae"
---

# TASK-0069 HANDOVER

## 成果

- `dev-agent-launcher`をexact CLIから一つのcoding-agentセッションを起動する実装へ変更した。
- fixed control socketから公開CA、GitHub REST/OpenAI用Opaque handleを取得し、loopback bridge、限定environment、Git helper、子processへ一回のlifecycleとして束縛した。
- partial setup、子終了、cancel、bridge failureの全経路でbridge drain、CA directory removal、handle revokeを所有し、固定診断とnon-leakを維持した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/launchsession ./internal/command` | `PASS` |
| harness `make check` | `PASS`（live testは既定`SKIP`） |
| harness `make distcheck` | `PASS` |
| configure/install stagingとlink-time path確認 | `PASS` |
| candidate worktree `make lint-docs` | `PASS` |
| Main root Task checker | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み7パス、1,197 additions / 2 deletions、計1,199 changed lines。
- CLIは`run --repository owner/repo -- COMMAND [ARG...]`だけを受け、shellなしでargv/stdin/stdout/stderrを子へ渡す。socket/helper/provider/model/proxyは入力にしない。
- literal `/tmp`のfresh 0700 directoryと0600 regular CA fileを作り、固定IPv4 loopback bridgeと子を並行監督する。bridgeの予期しない終了又はcancelは子を停止・waitし、drain後だけlocal cleanupする。
- child environmentは`HOME`、`PATH`、`TERM`、`LANG`/`LC_*`、任意の`CODEX_HOME`だけを親から選び、Opaque API handle、proxy/CA、prompt無効化、system/global Git config無効化、helper reset後の固定Git command-scope configを一意に追加する。
- ordinary child exit 125とlauncher failureは`Result`で分離し、通常のchild exit codeを保持しながらlauncher failureだけを固定診断にする。

## 検証結果

- candidate gate root `make check`: `PASS`
- 最終実装/test bytesへのfocused race、候補READMEへのdocs lint、Task checker、diff check: `PASS`
- 最終candidate bytesへのharness check/distcheck/install staging: `PASS`

## 判断・既知の制約

- focused race初回FAILはfake childのchannel参照にhappens-beforeがないtest defectだった。fixtureだけを修正し、再実行はPASSした。
- candidate gate初回は候補READMEの用語lintをMain rootから事前観測できておらずcommit前に停止した。候補worktreeで自動修正可能な表記とfrequent prose termsを修正し、候補lintと二回目のcandidate gateはPASSした。
- DEV worktreeの`make task-check`はnode dependency取得時のregistry DNS failureで停止した。Main rootの既存依存を使うTask checkerはPASSし、製品FAILへ帰責していない。
- 初回Reviewのsignal findingはGo公式`os.ProcessState.Exited()`がUnix signal terminationで`false`を返す契約と照合し、誤検出に分類した。`HOME`経由のGit global config findingは有効と分類し、`GIT_CONFIG_NOSYSTEM=1`と`GIT_CONFIG_GLOBAL=/dev/null`を固定して最終candidateを再作成した。
- 実credential、実Git/`gh`/Codex/OpenAI、DNS/TLS、OS default-deny/network namespace、Unix socket permission/peer UID、systemd/VPSは未確認であり、live E2Eとして別途扱う。
