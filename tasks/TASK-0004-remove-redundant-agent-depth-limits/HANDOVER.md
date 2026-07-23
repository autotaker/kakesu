---
task_id: "TASK-0004"
status: complete
completed_at: "2026-07-14T21:44:08+10:00"
---

# TASK-0004 HANDOVER

## 成果

- 製品Task branch `task/TASK-0004-remove-redundant-agent-depth-limits`のcommit `32a1c38ace6b1395b82c5b001777355b459a9558`で、通常6 roleの重複したagent-local `max_depth = 1`を削除し、2-parent merge `514facbb461927c0e5fc376a56ab8f975c054940`として製品`main`へ統合した。
- project-scopedな`agents.max_depth = 2` / `agents.max_threads = 2`を全体トポロジー上限として維持し、Explorer固有の`max_depth = 0` / `max_threads = 1`と既存の明示launcher contractを維持した。
- canonical parser、digest、adapter render/checkが、通常roleへのlocal depth再導入とExplorer no-childの欠落・緩和を別の理由で同期前にfail closedするようにした。

## 主要な変更

- `.codex/agents/{main,planner,qa,reviewer,dev-luna,dev-sol}.toml`から`max_depth`だけを削除し、各roleの`max_threads = 2`、Explorer registry、model、effort、sandboxは変更していない。
- `scripts/task/agent-routing.mjs`に通常roleの`[agents]`構造検査を追加し、local `max_depth`を`ROUTING_ROLE_LOCAL_DEPTH_FORBIDDEN`、thread policy driftを`ROUTING_ROLE_THREAD_POLICY_MISMATCH`で拒否する。
- `scripts/task/agent-routing.test.mjs`に、通常6 roleのdepth key不在、全roleへの値を問わない再導入、Explorer depth/thread緩和、project depth/thread drift、adapter同期前fail-closedの決定的fixtureを追加した。
- `docs/development/agent-roles.md`を更新し、通常roleがproject-scoped上限に従うことと、Explorerだけがrole-local no-child上限を持つことを区別した。

## 検証結果

- `git diff --check main...HEAD`: pass。
- `node --test scripts/task/agent-routing.test.mjs`: 11件pass、0件fail。
- `node --test scripts/task/development-process.test.mjs`: 21件pass、0件fail。
- 製品`make check`: pass。Go build/test/vet、Python build/pytest/ruff、Rust build/test/fmt/clippy、tabletop・用語・文書検査、process tests 32件がすべて成功した。
- 検証後の製品Task worktreeはcleanである。
- 独立Reviewerは実装commit `32a1c38ace6b1395b82c5b001777355b459a9558`を再検査し、`pass`と判定した。
- 製品merge commit `514facbb461927c0e5fc376a56ab8f975c054940`は、第1親`a0661057b56461c1a0e0f01a326d487b094e5ea1`、第2親review済みcommit `32a1c38ace6b1395b82c5b001777355b459a9558`の正確な2-parent mergeである。
- generated work adapterはcanonical digest `5566794aaf22a890ef432e4e45b63a3a07e1bcb3eaf0cf620a47883c705c3445`と完全一致し、同期はno-opである。
- 独立QAは製品`main`のmerge commit `514facbb461927c0e5fc376a56ab8f975c054940`をQA PLAN revision 1で検証し、QA-001〜QA-007をすべて`pass`、未実施項目なし、受け入れを妨げる未解消事項なしと判定した。
- マージ後QAではrouting tests 11件、development-process tests 21件、製品`make check`、adapter complete-match、`make -C /Users/autotaker/git/agent-harness work-check`がすべてpassした。
- 最初のQA sandbox内`make check`はPyPI DNS制約で停止したが、同一gateを許可済みビルド環境で再実行してpassしたため、解消済みの環境所見であり製品acceptance failureではない。

## 判断

- 通常roleのchild到達可能性はproject `.codex/config.toml`だけを正本とし、role-local depth keyの不在自体をcanonical contractとして検査する。
- Explorerのno-childは通常roleと一括処理せず、Explorer TOML、固定prompt、read-only sandbox、明示launcher、closed stdin、write scopeなし、`commit:null`、child-spawn負例の多層contractとして維持する。
- canonical driftはdigest計算・adapter render/checkより前に拒否し、driftした設定からadapterを生成または更新しない。

## 既知の制約と未解決事項

- 受け入れを妨げる既知の制約または未解決事項はない。
- QA後の最終フェーズ判断はmain Agent、HANDOVERのWiki ingestはWiki Agentが所有する後続処理である。QA RESULTの判定は`pass`で、未実施項目はない。

## 運用上の注意

- DEVはproduct Task worktreeだけを変更した。work repositoryの`.codex/config.toml`はmain Agentがlock下の専用sync launcherで同期・commitする。
- product `main`にはTask外の未commit `.codex/config.toml`変更として`features.multi_agent_v2`の4行が残っている。TASK-0004のmerge commitには含まれず、DEV、Reviewer、main Agentはいずれも変更・stage・commitしていない。マージ後QAでもTASK-0004のcanonical depth contractとこの既存差分を分離して評価し、変更しなかった。
- マージ後QAはproduct `main`のmerge commit `514facbb461927c0e5fc376a56ab8f975c054940`を対象にし、QA PLAN revision 1の期待結果と範囲を変更せずに完了した。

## Wikiへ引き渡す知識

### 再利用可能な知識

- 同じ深度keyでも、通常roleでは「keyが存在しないこと」、Explorerでは「`max_depth = 0`が存在すること」が別の不変条件になる。構造検査を分けることで、重複設定の再導入と権限境界の緩和を区別できる。
- canonical parserをdigest、adapter render、adapter checkの共通入口にすると、構造driftを生成・同期前に拒否し、adapterを変更前の状態に保てる。
- Explorer no-childは深度値だけに依存せず、固定prompt、明示launcher、read-only、closed stdin、write scopeなし、`commit:null`、child-spawn負例を重ねて検証する。

### 反例・失敗・注意点

- 通常roleとExplorerの`max_depth`を一括削除するとExplorer固有のno-child防御を失う。通常roleだけを列挙し、Explorerのrequired keyを別検査にする必要がある。
- projectの値だけを照合するテストでは通常roleへの重複key再導入を検出できない。各通常roleへの再導入をtemporary canonical fixtureで注入し、parser、digest、render、checkの全入口を確認する。
- Reviewerがproduct `main`を暗黙に対象にすると、未mergeのTask branch実装を見落として誤ったFAILになる。reviewed commitをlauncher入力と証跡で固定する。

### 更新候補ページ

- `wiki/semantic/schemas/codex-agent-model-routing.md`
- `wiki/semantic/scripts/task-delivery.md`

## ブートストラップ例外

- 該当なし
