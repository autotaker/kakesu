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
candidate_commit: "9b204317220a061a370d19337cf6fc225062539e"
candidate_tree: "45e1ad1f8038e3eecb1c619e8726a09ffb4f4d17"
managed_path_digest: "7da53973db0ee2e1f91148723d9c3db2a4fc23846a3ecbc32c09f797f2cb2d85"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
---

# TASK-0033 HANDOVER

## 成果

- 単一リポジトリのmain証跡正本、sparse Taskワークツリー、action別証跡トランザクション、PR/CI/post-merge/sync、固定REF-2移行validatorを実装した。
- Mainによるbootstrap、強いGit metadata quarantine、candidate固定は完了した。GitHub実環境確認、merge、archiveは未実施である。

## candidate-bound DEV証跡

- `candidate_commit`: `9b204317220a061a370d19337cf6fc225062539e`
- `candidate_tree`: `45e1ad1f8038e3eecb1c619e8726a09ffb4f4d17`
- `managed_path_digest`: `7da53973db0ee2e1f91148723d9c3db2a4fc23846a3ecbc32c09f797f2cb2d85`。DEV初回working-tree manifestは33ファイル、SHA-256 `77dcb2599590fb8ab78a101de7addc403811e797844cdc1ceec8ac8b6c9fcdaa`。
- `bootstrap_evidence_commit` / `bootstrap_evidence_digest`: `a063f6d461bbc6ce752d93306f83e4939e299d1e` / `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`。

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

- `project.yaml`、operations Schema、固定REF-2 migration manifest/verify/freeze/unfreezeを追加した。freezeは旧リポジトリのGit metadataを製品repositoryのGit common dirへ隔離し、unfreezeでmetadataと元`core.hooksPath`を復元する。
- `task-start`をclean/current main、証跡commit/push、sparse branch/worktree、allocation/publish失敗訂正を一括する入口にした。
- `evidence-commit`へ明示main root、action allowlist、共通lock、検査、完全staging、最大2回のfetch/rebase/revalidate/pushを実装した。
- 共通lockをworking treeの`.locks/`からGit common dirの`agent-harness-locks/`へ移し、bootstrap前の旧`.gitignore`でもlock自身が変更scopeへ混入しないようにした。
- composite candidate、PR scope、ready PRとmerge-commit auto-merge、read-only main CI、required PR checks、closed+merged post-merge、sync/FASTを実装した。
- 外部`WORK_ROOT`依存を削除し、関連Make入口、文書、テンプレート、用語集を単一rootへ更新した。

## 検証結果

- `make check`: 修正後PASS（93 process testsを含む）。初回DEV時のsandbox実行はisolated Python buildのDNS拒否でexit 2、同一差分をnetwork許可環境で再実行してexit 0。差し戻し修正後はcache済みsandboxでexit 0。
- `node --test scripts/task/unified-lifecycle.test.mjs`: 最終18/18 PASS。
- `pnpm test:process`: 最終97/97 PASS。既存のactive owner拒否、stale owner回復、rollback後lock解放検査もcommon-dir配置でPASS。
- `make lint-docs`: PASS。`git diff --check`: PASS。
- tabletop validator: 4 scenarios / 124 sequence payloads / 119 canonical payloads PASS、negative 11件 PASS。
- 固定REF-2 plan: source commit `d030db5dc2974056387616d047197823b94602ce`、tree `f5a5fde073836bc9965b9a05ae4ccf06f36eccaa`、historical 32、current 1、Task files 198、Wiki 29、Lap30 1。

### bootstrap transaction FAIL分類と修正

- 実行時FAIL: `Evidence scope violation for bootstrap: .locks/work-repository.lock/owner.json; transaction_start=e3f6da7`。
- 分類: `implementation_defect`。bootstrap対象の旧product mainには`.locks/` ignoreがなく、`acquireWorkRepoLock`後に`changedFiles`がlock ownerを未追跡製品pathとして検出した。commit、push、freezeは発生していない。
- 修正: `git rev-parse --path-format=absolute --git-common-dir`で共有Git metadata内のlockを解決する。排他、owner PID、stale復旧、release契約は維持した。
- 回帰証拠: `.locks/` ignoreを持たない旧main fixtureでbootstrap transactionがmanifestだけをcommitし、working treeに`.locks/`を生成しないnegative/positive検査を追加した。`scripts/task/lib.mjs` SHA-256 `ace486297db5216673612f9553a1225edf98dafe3f2e15f15ab961faa4d81431`、回帰test SHA-256 `bff5f3dcc647fbc6425fda54e3362a3965c9f3a22f87c94628c7ae45f54d89a1`。

### pre-merge evidence validation FAIL分類と修正

- 実行時FAIL: bootstrap/rebase/candidate固定後、main上のHANDOVER公開前に`make work-check MAIN_ROOT=/Users/autotaker/git/agent-harness`が`missing schemas/operations/{backlog,decision,ingestion-receipt,bootstrap-manifest}.schema.json`で停止した。`make task-check TASK=TASK-0033`はPASS。証跡commitは行っていない。
- 分類: `implementation_defect`。コードPR merge前のmainにはoperations Schemaがまだ存在しない一方、validatorは証跡データrootとSchema rootを同一に扱っていた。
- 修正: `validate-work`へ明示`--schema-root`を追加した。pre-mergeの`handover | review | qa-result` transactionだけは、main HANDOVERのcandidate commit/tree/managed digestとbootstrap binding、記録branch HEAD、clean worktreeを照合してからcandidate側validator/Schemaでmain証跡データを検査する。operations Schemaがmainへmerge済みなら常にmain自身のvalidator/Schemaを選び、candidate fallbackを使わない。
- 回帰証拠: mainからSchemaとvalidatorを除いたfixtureでcandidate-bound HANDOVER transactionがPASSし、その後mainへSchema/validatorを導入してcandidate Schemaを破損してもmain境界が選択されることを確認した。実main対象の`make work-check MAIN_ROOT=/Users/autotaker/git/agent-harness`も`Validated 2 epic(s), 33 task(s), and 14 Wiki page(s).`でPASS。validator SHA-256 `5c67324774498cc448b9fca48ec1c120bae23b99e0a278f0309b292bdd5b21e4`、transaction SHA-256 `b396cd64b216d3fe62a619c19b5d4e1d4df2f418f6cb1342248804a8d6b46e7a`、fixture SHA-256 `0a87b5279afbba36930ffbbc841c1329dbf84841cb76ed0c4eef04b0f8db9c4a`。
- sparse worktreeの製品checkがmain管理`tasks/**`/`wiki/**`を用語集入力として開こうとしないよう、これらをglossary scopeから除外した。移設前絶対パスを含むPython/Rust build cacheは再同期・clean後に再実行し、最終`make check`はexit 0。

### 独立REVIEW/QA FAIL分類と差し戻し修正

- R-001は`implementation_defect`。旧freezeは`pre-commit` hookだけで、`--no-verify`と低水準Git書込を拒否できなかった。修正後は、clean・単一worktree・期待HEAD・authorityを照合して旧repositoryの`.git`全体を製品repositoryのGit common dir配下へ隔離する。旧rootの通常Git discoveryを失敗させ、`git status`、`git commit --no-verify`、`git commit-tree`の全negativeを確認した。旧hook markerからのupgrade、隔離途中失敗時のpending cleanup、unfreeze途中失敗後の再実行、元`core.hooksPath`復元を実装した。Mainは旧repository HEAD `a49338d5013f8e54f72a9c7cc4f92c4a76c52d91`を照合してquarantineへupgradeし、旧rootの`git status`がnot-a-repositoryで失敗することを確認した。
- R-002は`implementation_defect`。main CIの`--allow-merge true`が任意の二親commitを許容していた。修正後は第2親をfirst-parent時点の単一Task HANDOVERへ一意に束縛し、candidate commit/tree、managed-path digest、main管理path不在、bootstrap commit祖先関係、bound manifest自己digestをすべて照合する。HANDOVERに束縛されない二親mergeのnegativeと、正しく束縛されたmergeのpositiveを確認した。
- R-003は`implementation_defect`。旧`task-start`はremote evidenceをpublishしてからworktreeをallocateし、補償push失敗でremoteに片割れTaskを残し得た。修正後はbranch/sparse worktreeを先にallocateし、成功後だけevidence commit/pushを開始する。push拒否かつremote不変時は、追加の補償pushを行わず、invocationが作ったworktree、branch、local evidence commit、Task directory、backlog assignmentを全て除去する。remoteが開始HEADから変化した場合は自動破棄せずreconciliation用に保持する。拒否hook fixtureでpush試行が規定の3回だけであることとremote不変を確認した。
- QA-003/QA-009は`implementation_defect`（証跡結合）。旧verifyは現在working treeとbootstrap manifestを比較したため、bootstrap後のappend-only HANDOVER更新を誤ってdigest mismatchにした。修正後は現在HANDOVERの`bootstrap_evidence_commit`/`bootstrap_evidence_digest`を読み、そのimmutable commit tree内のmanifest、全entry、`project.yaml`、manifest自己digestを検証する。binding後に現在のLap30/HANDOVERが変化してもPASSし、bindingを変更後commitへ差し替えるとFAILするnegativeを追加した。実mainの`make bootstrap-verify`は`279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`を返してPASSした。
- 最終検証: `make check` exit 0（process 97/97、memory 20/20、Go/Rust/Tabletop/terminology/docs lintを含む）。初回sandbox実行はPython build dependencyのDNS拒否でexit 2と分類し、network許可付き同一差分で再実行後PASS、最終cache済みsandbox再実行もPASSした。実main対象の`make task-check TASK=TASK-0033`、`make work-check MAIN_ROOT=/Users/autotaker/git/agent-harness`、`make bootstrap-verify MAIN_ROOT=/Users/autotaker/git/agent-harness`はいずれもexit 0。`git diff --check`もPASSした。
- 修正後SHA-256: `Makefile` `dc7079daa7a69842b83fa2d11e4780c3212c73ec4e20edccd787e6fa449e5e34`、`migrate-operations.mjs` `c74f85dd7642b2f7d429527a1ca9da3ff57a325f175f8bfbe7f89c5cb8c57ad9`、`unified-lifecycle.mjs` `93ffe98c687e3b8a9e70e0f763796e3e9dcd7625cc4520d6cfea1ede00ebb76e`、`unified-lifecycle.test.mjs` `01142f53b06d12fdae7ac86953e5a37a3cbbfec7c969572091ddf9e9adca336e`。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- 選択: `full-rerun`（candidate固定済み。carry-forward不可のsecurity/workflow/schema/lifecycle変更）。
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: candidate `9b204317220a061a370d19337cf6fc225062539e` / tree `45e1ad1f8038e3eecb1c619e8726a09ffb4f4d17` / managed digest `7da53973db0ee2e1f91148723d9c3db2a4fc23846a3ecbc32c09f797f2cb2d85`、bootstrap `a063f6d461bbc6ce752d93306f83e4939e299d1e` / digest `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`。実PR run `29973308835`で存在しない`setup-uv@v8`と、base側だけの証跡更新をPR変更と誤認するtwo-dot scope比較を検出した。公式latest major `v9`へ更新し、PR eventだけmerge-base比較へ変更、diverged main evidenceのpositive/candidate evidenceのnegative fixtureを追加した。CI/Scope契約変更を含むためcarry-forwardせず、独立REVIEW/QAを新しい同一composite candidateから全面再実行する。
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable | complete | incomplete`
- 影響QAケース集合が空でない場合の再実行証拠: TODO
- `merge_tree`と案 treeの比較: `pending`

## 既知の制約と未解決事項

- bootstrap後の`make task-check TASK=TASK-0033`と、candidate Schemaを明示する`make work-check MAIN_ROOT=/Users/autotaker/git/agent-harness`はPASS。candidate更新後にMainがbindingを再固定し、証跡transactionを再実行する。
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

- source freezeをhookだけにすると`--no-verify`や低水準Git操作で回避できる。旧`.git` metadata全体を新正本のGit common dirへ隔離し、旧rootのGit discoveryをfail-closedにする。rollback用markerは旧rootに残し、元`core.hooksPath`も保存する。
- bootstrap証跡commitは製品変更merge前なので、検証器を古いmain treeから読まず、承認済みTaskワークツリーのmigration verifierで検査する。

### 更新候補ページ

- `wiki/semantic/concepts/development-task.md`
- `wiki/decisions/DECISION-0004-multiagentv2-role-startup.md`

## ブートストラップ例外

- TASK-0033だけは既存外部証跡を固定REF-2 + current Task overlayとして製品mainへ直接bootstrapする。これはPR scope例外ではなく、bootstrap後のコードbranchは通常どおりmain管理pathを含めてはならない。
