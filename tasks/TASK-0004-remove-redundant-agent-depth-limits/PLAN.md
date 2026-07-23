---
task_id: "TASK-0004"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-14"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "canonical TOML、parser/digest/drift検査、生成adapter、Explorerのfail-closed権限境界、fixtureを横断する設定契約変更であるため"
approved_dev_profile_risk_signals:
  - cross_cutting
  - configuration_contract
  - sandbox_commit_ownership
planned_implementation_files: 7
planned_implementation_lines: 30
estimate_points: 3
---

# TASK-0004 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| 通常6 role に agent-local な深度上限がない | canonical parser の正常 fixture が main / planner / qa / reviewer / dev-luna / dev-sol で `[agents].max_depth` の不在を検査し、各 TOML を直接照合する。 | product `.codex/agents/{main,planner,qa,reviewer,dev-luna,dev-sol}.toml` |
| 深さ上限の正本は project-scoped のまま | canonical parser と delegation fixture が project `.codex/config.toml` の `agents.max_depth = 2`、`max_threads = 2`、depth 1/2 許可と depth 3 / thread 3 拒否を検査する。 | `agent-routing.mjs` の `readCanonicalContracts` / `validateDelegation` |
| Explorer だけが no-child を保持する | Explorer TOML の `max_depth = 0` / `max_threads = 1` と固定 prompt を parser が必須検査し、Explorer child chain は `ROUTING_EXPLORER_SPAWN_FORBIDDEN` で拒否する。 | `explorer.toml`、`validateDelegation`、既存 launcher fixture |
| 設定の再導入・緩和を同期前に検出する | 通常 role への `max_depth` 再導入、Explorer の `max_depth = 0` 削除・変更をそれぞれ一時 canonical fixture で fail closed にし、adapter check も canonical parser を経由して失敗することを検査する。 | `readCanonicalContracts`、`canonicalDigest`、`syncWorkAdapter(..., check: true)` |
| Explorer の明示 launcher contract を回帰させない | `run-explorer-agent.mjs` の spawn stub で fixed Luna/medium、read-only、closed stdin、write scope なし、`commit:null` と一問入力を検査する。 | TASK-0002 approved QA PLAN rev.2 の QA-004 / QA-005 |
| 文書が現行正本を正確に説明する | 文書レビューと検索で、通常 role の local depth 2 という旧説明を除き、project depth/thread 2 と Explorer 固有 no-child を区別する。 | `docs/development/agent-roles.md` |

## 関連Wikiと判断

- [Codex Agent Model Routing](../../wiki/semantic/schemas/codex-agent-model-routing.md) と [DECISION-0003](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md) に従い、project-scoped TOML を正本とし、Explorer は明示 launcher、fixed prompt、read-only、closed stdin、負例試験、trace による多層 no-child contract を維持する。通常 role の重複値をこの Explorer 境界の根拠にしない。
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md) と [DECISION-0001](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md) に従い、本 PLAN 承認後に実装前 QA 計画、独立 review、main merge、merge 後 QA の順を維持する。FAIL の差し戻し先は main Agent が決める。
- [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md) に従い、fixture / toolchain / sandbox の実行不能は環境所見として分離し、設定・実装の不一致を自動的に DEV へ帰責しない。
- TASK-0002 の approved PLAN と approved QA PLAN revision 2 を baseline とする。特に Explorer は custom-agent delegation ではなく `run-explorer-agent.mjs` の明示 launcher で実行されるため、depth/thread fixture と launcher contract fixture を混同しない。

## 設計

### 選択案

通常 role の child 到達可能性は、各 role file に複写する相対深度ではなく project `.codex/config.toml` の `agents.max_depth = 2` が一元的に規定する。main、planner、qa、reviewer、dev-luna、dev-sol の `[agents]` block から `max_depth = 1` の行だけを削除し、`max_threads = 2` と Explorer registry を変更しない。

`readCanonicalContracts` を構造検査へ拡張する。通常6 role に `[agents]` 内の `max_depth` key があれば、値にかかわらず専用の routing configuration error で拒否する。Explorer では従来どおり `max_depth = 0` と `max_threads = 1`、固定 prompt の no-spawn 文言を必須とする。project config は `max_depth = 2` と `max_threads = 2` を必須とする。このため、通常 role の深度値再導入と Explorer の no-child 緩和を別の不変条件として判定できる。

canonical digest と generated work adapter は既存どおり `readCanonicalContracts` を入口にする。よって、canonical role file の構造 drift は digest 計算・adapter render・adapter check の前に fail closed になる。adapter の生成形式（project の depth/thread 値と role registry）自体は変更しない。製品 `main` への統合後、main Agent が所有する `make work-config-sync` により work adapter を同期・別 commit し、DEV は work repository の `.codex/config.toml` を編集しない。

### 代替案と不採用理由

- 通常 role の `max_depth = 1` を残して値だけを project 値と合わせる案は、二つの正本を残し drift と起動拒否の曖昧さを解消しないため不採用。
- Explorer の `max_depth = 0` も一括削除する案は、Explorer 固有の no-child 防御を緩和するため不採用。
- launcher の prompt または `validateDelegation` だけで no-child を保証する案は、role configuration の drift を検出できないため不採用。role TOML、parser、launcher、負例試験の多層検査を維持する。
- generated work adapter に role-local depth を複写する案は canonical source を増やすため不採用。adapter は project settings と canonical role file の registry だけを決定的に参照する。

### 責務と境界

- product repository は canonical `.codex`、routing parser、fixture、製品文書を所有する。運用 repository は TASK / PLAN / QA / review / handover の証跡だけを所有する。
- DEV Agent は承認済み `sol-high` profile で product Task worktree のみを変更し、work adapter を編集・同期・commit しない。cross-cutting configuration contract のため Luna を選ばない。
- main Agent は PLAN 承認、work adapter の lock 下同期・commit、merge、QA FAIL 分類を所有する。Planner はこの PLAN のみ、QA は QA_PLAN / QA_RESULT のみを更新する。
- Explorer は明示 launcher による一問の read-only 調査だけを行う。generic delegation、fan-out、child spawn、Git / file write、scope expansion authority は持たない。

### 不変条件

- `agents.max_depth = 2` と `agents.max_threads = 2` は product project config にだけ存在する全体 delegation topology の上限であり、root→Explorer と root→role→Explorer は許可、depth 3 と3 thread以上は拒否する。
- main、planner、qa、reviewer、dev-luna、dev-sol は agent-local `max_depth` key を持たない。`max_threads = 2`、model、effort、sandbox、Explorer registry は従来値のままにする。
- Explorer は agent-local `max_depth = 0` と `max_threads = 1` を持ち、fixed prompt、explicit launcher、read-only sandbox、closed stdin、write scopeなし、`commit:null` と child-spawn負例試験の全てで no-child を維持する。
- canonical parser、digest、adapter render/check は、通常 role の prohibited key と Explorer の required key を同一視せず、いずれの canonical drift も生成・同期前に拒否する。

### 失敗時・移行・互換性

- canonical config の検査に失敗した場合は、launcher / adapter sync を開始せず nonzero で終了する。通常 role への key 再導入、Explorer key の欠落・非ゼロ化、project depth/thread の変更は各々 deterministic fixture で拒否する。
- 既存 work adapter は形式上 project depth/thread と role registry を維持するため、product merge 後に main Agent が `make work-config-sync` を実行して digest 付き完全一致を再確立する。sync / hook / post-check が失敗した場合は main-owned launcher の既存 rollback / `commit:null` contract に従い QAへ進めない。
- compatibility は fixed model / effort override、legacy Wiki route、parent-owned commit、TASK-0003 の未承認 native Explorer 計画に変更を加えない。live Codex の可用性は acceptance の決定的根拠にしない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `../agent-harness/.codex/agents/main.toml` | configuration | 1 | `max_depth = 1` だけを削除する。 |
| `../agent-harness/.codex/agents/planner.toml` | configuration | 1 | 同上。 |
| `../agent-harness/.codex/agents/qa.toml` | configuration | 1 | 同上。 |
| `../agent-harness/.codex/agents/reviewer.toml` | configuration | 1 | 同上。 |
| `../agent-harness/.codex/agents/dev-luna.toml` | configuration | 1 | 同上。 |
| `../agent-harness/.codex/agents/dev-sol.toml` | configuration | 1 | 同上。 |
| `../agent-harness/scripts/task/agent-routing.mjs` | implementation | 24 | 通常roleの prohibited local-depth と Explorer の required no-child depth を別々に fail-closed 検査する。 |
| `../agent-harness/scripts/task/agent-routing.test.mjs` | test | - | 正常系、通常role再導入、Explorer緩和、project depth/thread、adapter check、launcher contract の fixture を更新する。 |
| `../agent-harness/docs/development/agent-roles.md` | documentation | - | project-scoped topology と Explorer 固有 no-child の正本を区別する。 |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

実装/configuration は7ファイル、30行と見積もる。`file_score = ceil(7 / 3) = 3`、`line_score = ceil(30 / 200) = 1` のため、許容値 `1, 2, 3, 5, 8, 13` のうち最大値以上の最小値は3である。テストと文書は規則どおり見積もり対象外である。

## 実装手順

1. product canonical config と6通常role TOMLを確認し、project の depth/thread 2、Explorer の depth 0/thread 1、通常roleの `max_threads = 2` を保ったまま6行だけを削除する。
2. `readCanonicalContracts` に通常roleの local `max_depth` 不在を要求する構造検査を加え、Explorer の required depth 0 検査と project depth/thread 2 検査を明確に分離する。adapter render/check がこの parser を必ず通ることを維持する。
3. routing test に、6通常roleの正常不在、任意通常roleへの `max_depth` 再導入の拒否、Explorer depth 0 の欠落・変更の拒否、project depth/threadとdepth 3 / Explorer child / fan-outの既存拒否、adapter check の fail-closed を追加・更新する。
4. Explorer launcher の fixed CLI、single bounded question、read-only、closed stdin、write scopeなし、`commit:null` の既存 fixture を維持し、実 model 応答に依存しないことを確認する。
5. `agent-roles.md` を、通常roleは project-scoped topology に従い、Explorerだけが role-local no-child を持つ現行contractへ最小更新する。
6. product unit tests と `make check` を実行する。product `main` merge後、main Agent が `make work-config-sync`、adapter complete-match check、`make -C ../agent-harness work-check` をlock下で実行・記録する。

## 検証計画

- `node --test scripts/task/agent-routing.test.mjs`：canonical normal roles に local depth key がないこと、再導入を拒否すること、Explorer `max_depth = 0` / `max_threads = 1` の欠落・緩和を拒否すること、project depth/thread 2 と depth 3 / Explorer child / 3 thread / multi-question の正負例を確認する。
- 同 test の temporary canonical / adapter fixture：通常role key再導入またはExplorer key変更時に `readCanonicalContracts`、digest、`renderWorkAdapter`、`syncWorkAdapter(..., check: true)` が同期前に fail closed となることを確認する。adapter text は project `max_depth = 2` / `max_threads = 2` と registry を保つことを確認する。
- 同 test の Explorer launcher stub：固定 `codex exec` arguments、Luna/medium、read-only、`stdio:["ignore","pipe","pipe"]`、empty write scope、one-line launch evidence、`commit:null` を確認する。
- `make check` と `node --test scripts/task/development-process.test.mjs`：既存 routing / parent-owned commit / closed stdin / hook / rollback regression を確認する。
- product merge後に main Agent が `make work-config-sync`、`make work-config-sync CHECK=1`、`make -C ../agent-harness work-check` を実行する。shared hook、adapter sync、main-only commit が外側の lock / 権限により実行不能なら、QA FAIL Attribution に従って環境所見として別記し、PASSへ読み替えない。

## 未解決事項

- なし。Explorer は本 planner 実行で許可された一問の read-only launcher を呼んだが、sandbox が Codex state DB / in-process client の初期化を拒否して `CODEX_EXIT_1` となった。これは live Explorer の環境所見であり、上記の決定的な parser / spawn-stub fixture の受け入れ期待を変更しない。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 設計観点と代替案を検討している。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV profile `sol-high`、選定理由、risk signalsを承認した。
- [x] DEV開始を承認した。
