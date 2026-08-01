---
task_id: "TASK-0059"
change_class: "safety_contract"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T22:18:33Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0059 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。対象は開発統制の起動・所有権・任意receipt契約だけであり、製品成果物、製品test、runtime/build設定、Schema、依存、生成入力/成果物、外部観測可能な製品挙動は変更しない。

標準role registryと既存process testを用いて、削除対象のlauncher/Make入口が残らないこと、`wiki` roleと唯一の標準spawn経路、親Mainだけのscope/validation/lock付きcommit所有、receiptの任意性を確認する。これは安全契約変更であり、製品QA_RESULT/PASSおよびlive E2Eは計画しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | candidate treeの実行・設定pathを検索し、`scripts/task/run-wiki-agent.mjs`の削除、`legacy-wiki`、Wiki専用`codex exec`、`WIKI_PROFILE`/`WIKI_MODEL`/`WIKI_EFFORT`、`wiki-context`/`wiki-ingest` Make入口の不在を確認する。Task証跡内の削除理由は除外する。残存reference、削除されたlauncherを再導入するwrapper、Wiki以外の既存Make入口の削除をnegativeとして検出する。 | `focused-rerun` / repository source/process fixtureとbounded searchで、外部環境なしに削除契約を決定的に検出できる。 |
| QA-002 | AC-2 | `.codex`正規registryの新規`wiki` roleがTerra/medium、workspace-write、編集専用、子Agent起動禁止、Git書込禁止を宣言することを確認する。規約とrouting/process testが、Mainの `agents.spawn_agent(task_name=..., agent_type="wiki", fork_turns="none", ...)` だけを起動経路とし、独立launcher、異なるagent_type、fork省略、Wiki子による別Agent起動を拒否/検出することを確認する。Planner/DEV/Reviewer/QA/Explorerのrole/model契約が変化しないことも確認する。 | `focused-rerun` / role TOMLとprocess fixtureはhermetic・deterministic・boundedである。 |
| QA-003 | AC-3 | `AGENTS.md`、Wiki規約、Agent責務、unified lifecycle/process sourceとtestを照合し、親Mainが起動前にTask/scope/許可pathを固定しwriterを直列化すること、子終了後にscope、索引変更時だけ`make wiki-index`、`make work-check`を確認すること、既存共通lock付きMain publish transactionだけがcommitを所有することを確認する。Wiki子がlock/validation/stage/commit/merge/`.git`書込を所有する経路、同時writer、親のscope未確認、index変更なのに索引確認なしをnegative fixtureで検出する。 | `focused-rerun` / Node process fixtureとsource auditでMain/child責務と禁止経路を外部副作用なしに確認できる。 |
| QA-004 | AC-4 | Wiki依頼なしのTaskがWiki Agentもreceiptもなしで完了できること、receipt Schemaと既存receipt検証が残ること、receipt作成がcompletion gate・暗黙処理・必須launcherへ戻らないことを既存process testとsource auditで確認する。receiptの削除/書換え、Schema変更、完了条件へのreceipt追加、暗黙Wiki spawnをnegativeとして検出する。 | `focused-rerun` / process fixtureと静的契約はhermetic・boundedであり、実Wiki ingestは要求しない。 |
| QA-005 | AC-5, AC-6 | launcher固有fixtureが削除され、残るfocused process testsがQA-001〜004の不在、標準role、Main所有、receipt任意性を失敗検出することを確認する。candidate diffが許可11 pathsに限定され、`check-task.mjs`は必要な統制pathだけをexactに許可し、TASK-0058の既存Wiki差分・既存receipt/Decision・Wiki Schema・製品pathに遡及変更がないことを確認する。実装担当が実行した影響Node tests、docs lint、work-check、root `make check`、`git diff --check`の同一candidate証跡をcommand/cwd/resultまで独立監査し、QAはroot full checkを重複実行しない。 | `evidence-review` / candidate diff、focused test本文とcandidate-bound実装証跡で統制差分・非製品性を監査できる。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜004のprocess/source checksを次の一回だけ実行する。test pathまたはfilterはcandidateで実際に削除不在・role契約・親所有・receipt任意性の少なくとも一つを壊すと失敗する必要がある。

```sh
node --test scripts/task/development-process.test.mjs scripts/task/unified-lifecycle.test.mjs
```

zero exitだけでは不十分であり、対象testの欠落、skip、弱体化、candidate不一致、またはnegative scenarioの欠落は該当ケースをPASSにしない。`make wiki-index`、`make work-check`、docs lint、root `make check`、`git diff --check`はQA-005のDEV evidence review対象であり、このfocused rerunに加えてQAが再実行しない。

## 境界・異常・回帰

- Wiki Agentは編集のみであり、Main指定外のWiki path、stage/commit/merge、`.git`書込、lock/validation、子Agent起動を行えない契約を維持する。親Main以外へのcommit所有権の移動はFAILである。
- Mainの共通lock付きpublish transaction、scope確認、索引変更時の`make wiki-index`、`make work-check`を専用launcherや新wrapperで置換しない。暗黙起動、外部送信承認、new model/profile/effort variable、new checklist/versionを導入しない。
- receiptは作る場合だけ既存Schema/検証を使う任意成果物であり、receiptなしの完了を妨げない。既存receipt/Decision、Wiki本文、Schema、TASK-0058済み差分の変更はscope逸脱である。
- 許可path外、製品コード/test、runtime/build設定、dependency、生成物、外部観測可能な製品挙動への変更は `implementation_defect` または `requirement_gap` として分類する。failureをDEV不具合と決めつけない。

## 実装後の再確認

- [ ] 同一candidateでQA-001〜005を独立に評価し、指定focused process testsを一回だけ実行した。
- [ ] launcher/legacy/Make入口不在、standard wiki role、Mainだけのscope/validation/lock付きcommit所有、receipt任意性のnegative failure-detectionを確認した。
- [ ] DEV evidenceとしてNode tests、docs lint、work-check、root `make check`、diff checkを監査し、QAはfull checkを重複実行していない。
- [ ] 製品成果物不変、TASK-0058既存差分保持、製品QA_RESULT/PASSおよびlive E2Eを計画していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立安全契約QA計画。標準Wiki role、Main所有、receipt任意性、focused process testsを定義。 | `approved` |
