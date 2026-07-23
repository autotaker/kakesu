---
task_id: "TASK-0032"
status: complete
completed_at: "2026-07-22T08:51:04+10:00"
safety_checks:
  process_tests: pending
  contract_scope: pending
  docs_lint: pending
  make_check: pending
safety_checked_at: ""
safety_check_digest: ""
safety_candidate_tree: ""
safety_merge_tree: ""
---

# TASK-0032 HANDOVER

## 成果

- 旧`[features.multi_agent_v2]`、`hide_spawn_agent_metadata`、`tool_namespace`、明示的な`[agents]`ヘッダー、全体`max_threads`/`max_depth` overrideを製品configと生成アダプターから除去した。
- ユーザーclarificationに従い、製品configと生成アダプターの`main`、`planner`、`qa`、`reviewer`、`dev-luna`、`dev-sol`、`explorer`の7件の`[agents.<role>]`定義と`config_file` mappingを維持した。
- fallback delegation検査をCodex組み込み既定のdepth 1 / threads 6へ合わせ、旧設定、明示parent header、全体override、role欠落・追加・mapping変更をadapter同期前にfail-closedするfixtureを追加した。

## candidate-bound DEV証跡

- `candidate_commit`: `513b25001c595978234f9f502036b88f0be346b8`
- `candidate_tree`: `aff105a53fed0c1f71f709ef6fd54439f7340d1c`
- candidate worktree: `/Users/autotaker/git/agent-harness-work/worktrees/TASK-0032-reset-default-agent-config`
- 実行環境: Darwin 25.5.0 arm64、Node.js v24.1.0、Python 3.12.3

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001 | `node --test scripts/task/agent-routing.test.mjs`（canonical config静的検査、旧feature/key再導入negative） | candidate worktree、`canonicalFixture()`の一時`.codex`複製、`routing-drift-adapter-*` | Node test cacheなし、lockfile固定済みlocal `node_modules`、外部networkなし | 0 | `.codex/config.toml` SHA-256 `b0f2fbab262cea6fd9f203022b74868db5dafa1c3a715e3c7f94494772dfb275` | なし |
| QA-002 | `node --test scripts/task/agent-routing.test.mjs`（header/global default境界、7 role/mapping保持） | 同上。root→Explorer、nested depth 2、threads 6/7境界fixture | 同上 | 0 | config SHA-256 `b0f2fbab...b275`、canonical digest `3f72485c36b1deba725b476efed7e4969a34558226539be753eb3264a76e354e` | なし |
| QA-003 | `node --test scripts/task/agent-routing.test.mjs`（`renderWorkAdapter`、clean/drift `syncWorkAdapter(..., check: true)`） | `routing-adapter-*`一時directory。実運用configは書き込まない | Node test cacheなし、deterministic temporary fixture、外部networkなし | 0 | generated adapter SHA-256 `1482efbc34914b18a08a39764b089eee337c651ebf7d38b49bbee88d10efed14`、7 role、明示header/global override/旧featureなし | なし。実運用syncはQA-006へ分離 |
| QA-004 | `node --test scripts/task/agent-routing.test.mjs`; `shasum -a 256 .codex/agents/*.toml` | candidateの7 standalone role TOML、fixed role/Explorer/launcher fixture | Node test cacheなし、role TOMLは候補からread-only | 0 | 7 role file SHA-256は下記一覧、canonical digest `3f72485c...54e` | なし |
| QA-005 | `make test-process`; `make lint-docs`; `node scripts/build-tabletop-viewer-data.mjs`; `node scripts/validate-tabletop-scenarios.mjs`; `node scripts/test-tabletop-validator.mjs`; `make check`; `make task-check TASK=TASK-0032`; `git diff --check` | candidate worktree、process temporary Git/config fixture、12 viewer文書、4 tabletop scenario | final runは既存`.build/go-cache`、`.build/uv-cache`、Cargo target、lockfile固定local dependenciesを使用 | 0 | candidate tree `aff105a53fed0c1f71f709ef6fd54439f7340d1c`、product binary diff SHA-256 `9e5c90c06d35e6fe129745955740a8d11e5521243c2d6b6a85f522e1a524eb2d` | なし |
| QA-006 | merge後`make work-config-sync`; `make work-config-sync CHECK=1`; product/work mapping照合; Codex reload/restart後runtime確認 | Main所有の共通lock、承認済み隔離実環境、rollback/cleanup必須 | N/A | N/A | pending | 製品merge、運用adapter commit、承認済みruntime環境に依存するため未実施。Main所有の`live-e2e` |

- QAへ渡すネガティブ検出証拠: `Explorer and prohibited project settings fail closed before adapter sync`はExplorer no-child欠落/緩和、旧feature/key、明示`[agents]`ヘッダー、`agents.max_threads`/`agents.max_depth` dotted override、role欠落、mapping変更、未知role、top-level model driftを注入し、canonical parse/digest/render/checkの全入口で拒否する。`work adapter generation is deterministic and detects drift`はclean CHECK成功後に生成adapterのrole欠落、mapping変更、末尾driftをそれぞれ拒否する。delegation fixtureはdepth 1とthread 6を許可し、nested depth 2とthread 7を拒否する。
- テスト削除なし。ユーザーclarification前の「registry不在」期待を撤回し、7 role/mapping維持の正負検査へ置換した。candidate product binary diff SHA-256: `9e5c90c06d35e6fe129745955740a8d11e5521243c2d6b6a85f522e1a524eb2d`。
- standalone role file SHA-256: `dev-luna` `116dbd09927a90ca6a290b944edf5b16bd3a55895ac7d680510ca54ab26435bf`、`dev-sol` `f4d5c74fc42f0fdd7c6551ad096c796cf0785edce2aab25f1ad49f42cb1cccf0`、`explorer` `ac69678bdef1075a568511371d014874826bd795540cd39be7c00d3b41ec621a`、`main` `8caab008d12d301a42d3d2b0fb7a6034c7a676fcd3e818cfae02e566ccb82b78`、`planner` `f6a320ab2c4472bd89239765a01ec6f48e472cb529e5fbe1cd69c58ba8db1f21`、`qa` `2f0b35387932a2ac22a7a0b8ef9d5ed251dd39dcbe421d14666d9a99e9ea14ac`、`reviewer` `870ffea95d9bc66204e186053973521308ab5dbfb117a1d18e521ac29de11fc9`。

## 主要な変更

- `.codex/config.toml`: 旧feature block、明示`[agents]`ヘッダー、全体overrideだけを削除し、top-level 3値と7 child table/mappingを維持。
- `scripts/task/agent-routing.mjs`: canonical parserへ旧設定/global override/registry driftの構造検査を追加し、生成adapterへ7 role mappingを維持。fallback validatorをdepth 1 / threads 6へ更新。
- `scripts/task/agent-routing.test.mjs`: product/work registry保持、禁止設定再導入、role欠落/mapping変更、clean/drift CHECK、depth/thread境界を検査。
- `docs/development/agent-roles.md`と`docs/glossary.yml`: 明示parentなしのimplicit TOML parent、7 registry維持、組み込み全体上限を文書・用語inventoryへ同期。`.codex/agents/*.toml`、top-level model/effort/sandbox、role契約は変更なし。

## 検証結果

- routing test 11/11 PASS、process suite 79/79 PASS、`make lint-docs` PASS、tabletop build/validate/test PASS。
- `make check` PASS: Go build/test/vet、Python build/20 pytest/ruff、Rust build/test/fmt/clippy、用語、文書、process、tabletopの全gateがexit 0。
- `make task-check TASK=TASK-0032`と`git diff --check`はexit 0。
- merge前の実運用`make work-config-sync CHECK=1`はexpected driftをnonzeroで検出し、運用configを変更しなかった。temp fixtureでは同期済み出力のCHECKがexit 0、role欠落/mapping変更/末尾driftが全て拒否された。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- QA_PLAN revision 2のQA-001〜QA-005を同一candidateへ束縛した。ReviewerとQAは相互のPASSを開始条件にせず、このcommit/treeから独立評価する。
- 選択: `focused-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: candidate `513b25001c595978234f9f502036b88f0be346b8` / tree `aff105a53fed0c1f71f709ef6fd54439f7340d1c`を直接評価し、独立ReviewerとQA-001〜QA-005が同じ案でPASSした。merge commit `e3f6da75b597372db484ff722c2ccb414bf802fa`のtreeも同一であり、carry-forwardは使っていない。
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable`
- 影響QAケース集合が空でない場合の再実行証拠: QA-001〜QA-005をcandidate上でfocused-rerunし、QA-006をmerge後fresh runtimeで実施した。
- `merge_tree`と案 treeの比較: `pass` — いずれも`aff105a53fed0c1f71f709ef6fd54439f7340d1c`

## 既知の制約と未解決事項

- なし。QA-006は運用adapter commit `eb9a3ede41e4d1a18e57391c5d1d257be73d485a`、再CHECK、fresh Codex feature/config/role/depth観測によりPASSした。

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- 運用リポジトリの`.codex/config.toml`を手編集しない。製品merge後にMainが共通lock下で`make work-config-sync`を実行し、governance commit後に`CHECK=1`と`make work-check`を行う。
- merge後adapterは旧feature、明示`[agents]`ヘッダー、全体overrideがなく、7件の`[agents.<role>]`と各`config_file` mappingがproduct/workで対応することを確認する。

## Wikiへ引き渡す知識

### 再利用可能な知識

- TOMLでは`[agents.<role>]`がparent `agents`を暗黙に作る。全体overrideを除去する際、明示`[agents]`ヘッダーは不要だが、7 child tableはcustom subagent loadingのmappingとして維持する。
- canonical parserをdigest/render/checkの共通入口にすると、禁止設定の再導入だけでなくrole欠落・mapping変更も生成・同期前に拒否できる。

### 反例・失敗・注意点

- 「`[agents]`を削除する」を全child table削除と解釈するとcustom subagent定義を失う。明示parent/global keysと`[agents.<role>]` child registryを構造的に分けて検査する。
- merge前の実運用adapterは旧canonicalに一致するため、候補からの`CHECK=1` driftはexpectedである。DEVが同期せず、Mainがmerge後に専用launcherで更新する。

### 更新候補ページ

- `wiki/semantic/schemas/codex-agent-model-routing.md`
- `wiki/decisions/DECISION-0004-multiagentv2-role-startup.md`の後続知識として、明示parent/global overrideとchild registryの区別を取り込む。

## ブートストラップ例外

- 該当なし
