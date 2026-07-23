---
task_id: "TASK-0021"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 13
planned_implementation_lines: 1000
estimate_points: 5
---

# TASK-0021 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| Linux install/diagnostics | preflight/start/diagnoseのsupported/unsupported出力 | docs/07 §2, docs/13 §8 |
| full slice E2E | Task→grant ACK→manual retry→episode→context trace | docs/10 §12, docs/12 §§5-6 |
| restart recovery | 各Plane kill/restartでoutbox/inbox/lease convergence | docs/11 §5, docs/12 §7 |
| bounded release scope | test fixtureが新domain APIなしで既存契約だけ使用 | docs/12 §§2,10 |

## 関連Wikiと判断

- docs/07, 10-13。依存: TASK-0012、0013、0014、0015、0018、0020。Linux P0 CLI統合とrecoverabilityだけを所有する。

## 設計

### 選択案

Linux test harnessがlocal DNS/HTTPS fixtureと各Plane processを起動し、公開CLI経路だけで代表シナリオを駆動する。fault controllerはtransaction境界の直後にprocessを停止し、restart後に診断commandと永続record/messageを照合する。install/diagnoseは既存component healthとprofile preflightを集約する。

### 代替案と不採用理由

- unit testsの寄せ集め: process/namespace/outbox復旧を検証できない。
- productionネットワークE2E: 安定性・credential・費用を損なう。
- release時の新domain implementation: scopeと障害帰属を膨張させる。

### 責務と境界

- CLI/app: install/preflight/lifecycle/diagnosticsの集約。
- E2E fixture: local依存、公開API、fault injection/assertion。
- 各Plane: 既存のrecovery contractを提供するだけ。新policy/memory semanticsは変更しない。

### 不変条件

- Linux以外をsupportedと表示しない、external traffic/credentialなし、初回egressは自動再生なし、grant/episodeは最大一回確定、DBを直接編集せずCLI/message経路で検証、E2E失敗を自動で実装不具合と断定しない。

### 失敗時・移行・互換性

install preflight不足は非ゼロ終了と修復案を返す。restartは既存DB migrationを再実行可能にし、outbox/inbox/leaseが再配送で収束する。旧CLI環境変数は診断でdeprecationを示すが、MVP profile外を黙ってfallbackしない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/cmd/kakesu/{main,install,diagnose}.go | implementation | 180 | CLI/preflight/diagnostics |
| core/internal/app/{lifecycle,health,diagnostics}.go | implementation | 170 | process wiring/readiness |
| scripts/{install-linux,diagnose-linux,run-mvp-e2e}.sh | tooling | 150 | supported host workflow |
| test/e2e/{mvp_linux_test.go,fixture.go,fault_controller.go} | test | 350 | full slice/restart |
| docs/{operations/linux-mvp,development/mvp-e2e}.md | documentation | 150 | support/failure classification |

実装対象13ファイル・1,000行。`ceil(13/3)=5`、`ceil(1000/200)=5`のため5pt。

## 実装手順

1. Linux install/preflight/diagnoseとcomponent readinessを公開CLIへ追加する。
2. local fixtureで既存sliceを通す代表E2Eを構築する。
3. Plane別crash point/restart/reconciliation assertionを追加する。
4. unsupported/failure classificationとrunbookを整備する。

## 検証計画

- unsupported OS/capability、fresh install、start/stop/diagnose、root/subtask、grant deny/ACK/manual retry/one-use、episode/context、control/governance/memory停止、duplicate delivery/lease expiry、restart後のno duplicate effect。
- Linux E2E、各Plane integration、`make check`、QAで環境/仕様/実装/回帰の失敗分類を確認する。

## 未解決事項

- CI Linux runnerのnamespace/firewall権限とfault injection方法をQA_PLANで確定する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
