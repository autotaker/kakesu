---
task_id: "TASK-0033"
status: draft
completed_at: ""
safety_checks:
  process_tests: pending
  contract_scope: pending
  docs_lint: pending
  make_check: pending
safety_checked_at: ""
safety_check_digest: ""
safety_candidate_tree: ""
safety_merge_tree: ""
candidate_commit: ""
candidate_tree: ""
managed_path_digest: ""
bootstrap_evidence_commit: ""
bootstrap_evidence_digest: ""
---

# TASK-0033 HANDOVER

## 成果

- 単一リポジトリのmain証跡正本、sparse Taskワークツリー、action別証跡トランザクション、PR/CI/post-merge/sync、固定REF-2移行validatorを実装した。
- Mainによるbootstrap、candidate固定、GitHub実環境確認は未実施であり、DEVはコミット、stage、merge、freeze、archiveを行っていない。

## candidate-bound DEV証跡

- `candidate_commit`: Main固定待ち
- `candidate_tree`: Main固定待ち
- `managed_path_digest`: Main固定待ち。DEV working-tree manifestは33ファイル、SHA-256 `77dcb2599590fb8ab78a101de7addc403811e797844cdc1ceec8ac8b6c9fcdaa`。
- `bootstrap_evidence_commit` / `bootstrap_evidence_digest`: Mainのbootstrap実施待ち。

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001 | `rg -n 'WORK_ROOT|\.\./agent-harness-work|agent-harness-work' ...`; migration plan | Taskワークツリー、固定REF-2実リポジトリ | N/A | 0 | source tree `f5a5fde073836bc9965b9a05ae4ccf06f36eccaa`; plan digest（本HANDOVER追記前）`966ea7be65c663ff1d7ab37d39fed118d1ff6c6070d81757a781002f4ae69f4e` | なし。Mainは追記後にplanを再生成する |
| QA-002 | `node --test scripts/task/unified-lifecycle.test.mjs` | 一時Git/bare remote、成功・allocation失敗・publish失敗注入 | N/A | 0 | test file `bff5f3dcc647fbc6425fda54e3362a3965c9f3a22f87c94628c7ae45f54d89a1` | なし |
| QA-003 | 同fixtureのfreeze/unfreeze、allowlist、lock、retry上限、sparse検査 | 一時Git/bare remote | N/A | 0 | lifecycle `9a663b07ec3e2e198a3eaff21a7728237a0b7d57f407d04da7a74888678660b6` | 実bootstrap/freezeはMain待ち |
| QA-004 | workflow静的negativeと`make check` | ローカルfixture | `.build` cache使用 | 0 | main/pr/post workflow SHA-256: `55def6869b7a70bbabb75dfc2016e1392591366268287c90de090d3ed089138c`, `ebc683431449d75d15dfcbfc4997bfa605585953fa485d7c65b34c1f13c83ce1`, `d801ad9a982e625941e28e3d126e60d49a0c89887b3812657d5c1dd9e74eff56` | GitHub runはQA-006で確認 |
| QA-005 | 未実施 | 承認済みGitHub repositoryが必要 | N/A | N/A | N/A | bootstrap後のcomposite candidate、REVIEW/QA PASS、実authが必要 |
| QA-006 | 未実施 | GitHub ruleset/required checksが必要 | N/A | N/A | N/A | live-e2e |
| QA-007 | 未実施 | merged PR eventが必要 | N/A | N/A | N/A | live-e2e |
| QA-008 | FAST/no-op fixtureのみ実施。実Wiki取込は未実施 | fixture / 認証済みローカルCodex待ち | N/A | fixture 0 | test file `bff5f3dcc647fbc6425fda54e3362a3965c9f3a22f87c94628c7ae45f54d89a1` | live-e2eの取込、done化、cleanupはQA待ち |
| QA-009 | 固定REF-2 migration planとtamper negative fixture | 実source read-only + 一時target | N/A | 0 | 32 historical、1 current、229 entries、project digest `031b8315e0f96088be4efe8fc17cc018a77c779434599c8bf92b38ebc9d63a7f` | apply/commit/freezeはMain待ち |
| QA-010 | 未実施 | 公開GitHub repository archive権限が必要 | N/A | N/A | N/A | QA-009とcutover完了後のみ実施可 |

- QAへ渡すネガティブ検出証拠、テスト弱体化の有無を判定できる差分ダイジェスト: working-tree manifest `77dcb2599590fb8ab78a101de7addc403811e797844cdc1ceec8ac8b6c9fcdaa`。既存process testを削除せず14件のunified lifecycle fixtureを追加した。

## 主要な変更

- `project.yaml`、operations Schema、固定REF-2 migration manifest/verify/freeze/unfreezeを追加した。freezeは旧リポジトリの`core.hooksPath`を拒否hookへ切替え、unfreezeで元値へ戻す。
- `task-start`をclean/current main、証跡commit/push、sparse branch/worktree、allocation/publish失敗訂正を一括する入口にした。
- `evidence-commit`へ明示main root、action allowlist、共通lock、検査、完全staging、最大2回のfetch/rebase/revalidate/pushを実装した。
- 共通lockをworking treeの`.locks/`からGit common dirの`agent-harness-locks/`へ移し、bootstrap前の旧`.gitignore`でもlock自身が変更scopeへ混入しないようにした。
- composite candidate、PR scope、ready PRとmerge-commit auto-merge、read-only main CI、required PR checks、closed+merged post-merge、sync/FASTを実装した。
- 外部`WORK_ROOT`依存を削除し、関連Make入口、文書、テンプレート、用語集を単一rootへ更新した。

## 検証結果

- `make check`: 修正後PASS（93 process testsを含む）。初回DEV時のsandbox実行はisolated Python buildのDNS拒否でexit 2、同一差分をnetwork許可環境で再実行してexit 0。差し戻し修正後はcache済みsandboxでexit 0。
- `node --test scripts/task/unified-lifecycle.test.mjs`: 14/14 PASS。
- `pnpm test:process`: 93/93 PASS。既存のactive owner拒否、stale owner回復、rollback後lock解放検査もcommon-dir配置でPASS。
- `make lint-docs`: PASS。`git diff --check`: PASS。
- tabletop validator: 4 scenarios / 124 sequence payloads / 119 canonical payloads PASS、negative 11件 PASS。
- 固定REF-2 plan: source commit `d030db5dc2974056387616d047197823b94602ce`、tree `f5a5fde073836bc9965b9a05ae4ccf06f36eccaa`、historical 32、current 1、Task files 198、Wiki 29、Lap30 1。

### bootstrap transaction FAIL分類と修正

- 実行時FAIL: `Evidence scope violation for bootstrap: .locks/work-repository.lock/owner.json; transaction_start=e3f6da7`。
- 分類: `implementation_defect`。bootstrap対象の旧product mainには`.locks/` ignoreがなく、`acquireWorkRepoLock`後に`changedFiles`がlock ownerを未追跡製品pathとして検出した。commit、push、freezeは発生していない。
- 修正: `git rev-parse --path-format=absolute --git-common-dir`で共有Git metadata内のlockを解決する。排他、owner PID、stale復旧、release契約は維持した。
- 回帰証拠: `.locks/` ignoreを持たない旧main fixtureでbootstrap transactionがmanifestだけをcommitし、working treeに`.locks/`を生成しないnegative/positive検査を追加した。`scripts/task/lib.mjs` SHA-256 `ace486297db5216673612f9553a1225edf98dafe3f2e15f15ab961faa4d81431`、回帰test SHA-256 `bff5f3dcc647fbc6425fda54e3362a3965c9f3a22f87c94628c7ae45f54d89a1`。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- 選択: `not-applicable`（candidate未固定。carry-forward不可のsecurity/workflow/schema/lifecycle変更）。
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: candidate/bootstrap固定後にMainが記録する。
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable | complete | incomplete`
- 影響QAケース集合が空でない場合の再実行証拠: TODO
- `merge_tree`と案 treeの比較: `pending`

## 既知の制約と未解決事項

- `make task-check TASK=TASK-0033`と`make work-check`はbootstrap前の製品mainに運用証跡が存在しないため未実施。Mainはbootstrap commit後に実行する。
- QA-005〜QA-008の外部作用部分とQA-010はlive-e2e待ち。required checks/ruleset、auto-merge、post-merge event、実Wiki取込、archiveをfixture PASSで代替しない。
- candidate値とbootstrap値はMainのGit操作後に再束縛する。本HANDOVER追記でsource digestが変わるため、記載したplan digestをbootstrap値に流用しない。

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- MainはTaskワークツリーのコードを先にcommitしない。まずこのHANDOVERを含むsourceから`make bootstrap-plan`、`bootstrap-apply`、`bootstrap-verify`を行い、`ACTION=bootstrap`の証跡commit/push成功後にだけ旧sourceをfreezeする。
- bootstrap失敗時は旧sourceをfreezeしない。freeze後の失敗では先に`bootstrap-unfreeze`で旧authorityを復旧し、auto-mergeを無効化してからbootstrap revertとTask branch再作成を行う。
- bootstrap後、MainはTask branchを新mainへrebaseし、PR差分からmain管理証跡を消したうえでcandidate commit/tree/managed digestとbootstrap commit/digestを固定する。

## Wikiへ引き渡す知識

### 再利用可能な知識

- 証跡と製品を一つのGit履歴に置く場合も、main管理pathをsparseでコードワークツリーから除外し、証跡commitを明示main rootへ経路することで責務を分離できる。
- 切替bootstrapは固定source refと現在Taskだけのappend-only overlayを別々に照合し、manifestの自己digestと全file digestで再現可能にする。

### 反例・失敗・注意点

- source freezeをmarkerだけにすると旧hook導入前の時間帯にcommit可能になる。Git共通dirの拒否hookへ`core.hooksPath`を切替え、rollback用に旧値をmarkerへ保存する。
- bootstrap証跡commitは製品変更merge前なので、検証器を古いmain treeから読まず、承認済みTaskワークツリーのmigration verifierで検査する。

### 更新候補ページ

- `wiki/semantic/concepts/development-task.md`
- `wiki/decisions/DECISION-0004-multiagentv2-role-startup.md`

## ブートストラップ例外

- TASK-0033だけは既存外部証跡を固定REF-2 + current Task overlayとして製品mainへ直接bootstrapする。これはPR scope例外ではなく、bootstrap後のコードbranchは通常どおりmain管理pathを含めてはならない。
