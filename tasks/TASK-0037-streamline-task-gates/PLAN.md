---
task_id: "TASK-0037"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存のTask lifecycle/checker/templates/docsを削減・局所変更し、外部通信、Credential、OS権限、製品runtimeを変更せず、隔離Git fixtureで検証できるため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T02:08:37Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T02:08:37Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
planned_implementation_files: 7
planned_implementation_lines: 430
estimate_points: 3
---

# TASK-0037 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | planningは許可された変更済みpathの非空subsetを一度に公開し、4ファイルの最終状態だけを検証する。初回だけtaskStart生成の未追跡・空REVIEW/QA/HANDOVER scaffoldを同commitに含めてよいが、既存追跡済み証跡のforced dirtyは要求しない。 | lifecycle、checker、process test、Task/Plan templates、開発文書 | 1 | 空集合、許可外path、最終状態不整合、または既存追跡済みfileのforced dirtyはcommitしない。個別actionは復旧互換として残す。 |
| AC-2 | completionはmain側の未commit品質証跡HANDOVERからcandidateを読み、`git merge --no-ff --no-commit <candidate>`後の最終状態を検証して唯一のmerge commitを作る。candidateはplanning commitをfast-forwardしたbranch上のproduct-only単一commitとする。 | lifecycle、checker、process test、completion evidence templates、Git文書 | 2 | PASS、branch、merge-base..candidate count=1、product-only、第2親、scope、最終検証のいずれかが不正ならmerge abortし、main品質証跡のbytes/stageを復元してpartial staging/merge commitを残さない。複数candidate commitはMainがsquashして一度だけ固定する。 |
| AC-3 | 両gateはlock、scope、validation、commitを各一回に固定する。親がvalidation前に内容digestを固定し、hookは完全staging/no unstagedの同一digest時だけ再validationを省略する。 | lifecycle、lib、pre-commit、lifecycle test | 1–2 | digest不一致、欠落、通常commitはhook自身がvalidationする。 |
| AC-4 | completion前のmain側dirty HANDOVERの`candidate_commit`だけを候補の記録点とし、candidate branch/candidate commit自身にはHANDOVERを書かない。tree、製品diff、managed digest、merge commitはGitから導出する。 | lifecycle、checker、HANDOVER/REVIEW/QA templates、test | 2 | HANDOVER欠落、candidate branch不一致、導出値とmerge第2親の不一致、candidate側HANDOVER、自己参照/別evidence commitはcompletionを拒否する。 |
| AC-5 | reviewerは独立identity、PASS、candidate差分とDEV check証跡を監査した事実を記録する。QA focused rerunは独立して維持する。 | checker、REVIEW/QA templates、process test、AGENTS/development docs | 3 | reviewer identity/PASS/監査記録、またはcandidate整合が欠ければ拒否する。同一check再実行や単一値fieldは要求しない。 |
| AC-6 | estimateの算術完全一致をDone経路から外し、参考計測としてのみ残す。 | checker、process test、task-management docs | 3 | 見積不一致だけでDEVを停止しない。 |
| AC-7 | Wiki receiptを全Task必須から外し、再利用可能な知識がある場合だけcompletion transactionに含める。 | lifecycle、checker、test、Wiki/task-management docs | 3 | 知識なしはWikiなしで完了する。Wikiを含める場合のみallowlistで検証する。 |
| AC-8 | CF-1〜CF-7を除去し、candidate変更時は影響focused test、限定不能時だけfull rerunとする。 | QA templates、checker、QA docs/test | 4 | checklistによる旧QA結果の転用はしない。影響範囲に応じて再実行する。 |
| AC-9 | QA標準証跡をcase ID、HANDOVER candidate、command、結果へ縮小する。条件付き詳細は該当時だけ残す。 | QA/HANDOVER templates、checker、QA docs/test | 4 | 基本4項目が欠ける場合だけ拒否し、該当しない詳細を要求しない。 |
| AC-10 | model/effort mismatchは実測値を記録した警告に変え、identity・role・sandbox/権限境界の欠落だけを停止条件に残す文書契約へ更新する。 | `AGENTS.md`、`docs/development/agent-roles.md`、`scripts/task/development-process.test.mjs` | 4 | boundary不明または役割分離違反はfail-closed、単独のmodel/effort差は警告で継続する。新しいruntime checker/fieldは追加しない。 |
| AC-11 | task startはscaffold/branch/worktreeだけを原子的に準備し、最初のmain commitをplanning-gateにする。planning後にTask branch/worktreeをplanning commitへfast-forwardし、MainがDEV製品差分を一度だけcandidate commitにする。completionはその単一candidateを検査する。MakeはALLOW_MERGEを明示伝播し、標準3 commitを文書化する。 | lifecycle/test、Makefile、Git/process docs | 5 | taskStart失敗は作成物を復旧する。candidateが複数commitならMainがsquashして一度だけ固定する。allowなしはdirect product pathを拒否する。追加transactionは手戻り、復旧、live E2Eのみ。 |
| AC-12 | 規定検査を実施し、行数・見積・形式fieldを品質gateにしない。 | process tests、checker、全許可path | 5 | 実際の検査FAILのみを修正・再実行する。 |
| AC-13 | 文書を最小既定へ改め、rule/gateは反復failureまたは重大安全failureを直接防ぐ時だけ追加し、10 Taskごとの既存retrospectiveで低価値ruleを削除/警告化する。 | AGENTS、development docs | 5 | 単発軽微ミスへの恒久rule、専用Task/checklist/version/棚卸しcommitは作らない。 |

## 設計

標準経路は次の3 commitsとする。

```text
taskStart（scaffold / branch / worktree、main commitなし）
  → planning-gate main commit
  → branch/worktreeをplanning commitへfast-forward
  → Mainがproduct-only candidate commitを一度だけ作成
  → main側dirty HANDOVERでcandidate_commitを固定し、REVIEW/QAもmain側dirtyで作成
  → completion-gate no-ff merge commit
```

planning-gateは許可集合内の実変更だけをcommitし、PLAN、QA_PLAN、TASK、backlogの最終状態を読む。初回はtaskStart生成の未追跡・空REVIEW/QA/HANDOVER scaffoldを同commitに含められるが、既存追跡済み証跡をforced dirtyにしない。planning commit後はTask branch/worktreeをそのcommitへfast-forwardし、MainがDEV製品差分をproduct-onlyの単一candidate commitへsquashして固定する。HANDOVERはcandidate branch外のmain側dirty品質証跡としてcandidate commitを一度だけ記録し、REVIEW/QAもmain側dirtyで作る。completion-gateは`merge-base..candidate`が単一commitであることとproduct-onlyを検査してからcandidateを`--no-commit`でmergeし、候補第2親、REVIEW/QA PASS、post-implementation QA_PLANと最終状態を検証して一度だけcommitする。失敗時はmergeをabortし、開始前のmain品質証跡bytesとstage状態へ戻す。

candidateはcandidate branch外のmain側dirty HANDOVERの`candidate_commit`のみで束縛する。tree、managed digest、merge commit、tested ancestryは必要時にGit履歴から算出し、証跡ファイル間の転記、candidate側HANDOVER、自己参照の`merged_commit`/`tested_commit`、別evidence commitは削除する。

既存差分から削除する対象は、artifact/reviewのversion・mode・digest grammar、candidate tree/digestの重複frontmatter、estimate算術gate、全Task Wiki receipt、CF-1〜CF-7、全QAケースの詳細必須、model/effort mismatch停止、completion後の自己参照値である。追加scopeはplanning subset検証、atomic completion merge、optional Wiki、影響ベースのQA rerun、model/effort警告に限定する。

## 代替案と不採用理由

- 未変更の計画証跡を強制dirtyにする案は、検出力を増やさずcommitを増やすため不採用。
- evidence commitとcandidate mergeを分ける案は、completionの原子性と3 commit標準を失うため不採用。
- reviewerまたはQAを省略する案は、独立評価を失うため不採用。
- 単発の軽微な失敗ごとにfield、enum、checklistを追加する案は、検出価値より維持費を増やすため不採用。

## 変更予定

見積は計画上の参考であり、算術完全一致を機械gateにしない。

| ファイル群 | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `scripts/task/unified-lifecycle.mjs`, `lib.mjs`, `work-pre-commit.mjs` | implementation | 220 | planning subset、taskStart no-commit、completion no-ff transaction、digest handoff/復旧。 |
| `scripts/task/check-task.mjs` | implementation | 90 | Git導出、最小review/QA、不要Done gate削除。 |
| `Makefile` | implementation | 4 | ALLOW_MERGE伝播。 |
| `scripts/task/*process*.test.mjs` | test | 310 | atomicity、3 commits、候補/merge、削除契約の正負回帰。 |
| `templates/task/{HANDOVER,REVIEW_RESULT,QA_PLAN,QA_RESULT,PLAN}.md` | implementation | 90 | 最小candidate/review/QA証跡と不要field削除。 |
| `AGENTS.md`, `docs/development/*.md` | documentation | 160 | 3 commit、最小rule、optional Wiki、QA/model運用、10 Taskごとの既存retrospectiveへ更新。 |

## 実装手順

1. taskStartを非公開初期化に縮小し、planning-gateを最初のmain transactionへ変更する。
2. planning-gateのnon-empty subsetと最終状態検証を実装し、個別actionを復旧互換へ整理する。
3. planning後のbranch/worktree fast-forwardとMainのsingle product-only candidate squashを実装し、completionが`merge-base..candidate`のcount=1を検査する。
4. completion-gateのno-ff/no-commit、main側dirty品質証跡、最終検証、単一merge commit、abort/証跡bytes/stage復元を実装する。
5. HANDOVER唯一candidateとGit導出へ移し、candidate側HANDOVER、重複binding・形式/算術/Wiki/CF/自己参照gateをtemplates、checker、docs、testsから削除する。
6. QA/reviewer/model運用と最小rule/retrospective方針を文書化し、Make scope許可を修正する。棚卸しは10 Taskごとの既存retrospectiveで行い、専用Task/checklist/commitを作らない。
7. process、task、root checksを実行し、実際のFAILに限定して修正する。

## 検証計画

- lifecycle process tests: planningのnon-empty subset、最終状態、初回未追跡空REVIEW/QA/HANDOVER scaffoldの同commit許容、既存追跡済み証跡のforced dirty禁止、taskStart no-commit、planning後branch/worktree fast-forward、single product-only candidate、複数commit candidate拒否とMain squash責任、main側dirty HANDOVER/REVIEW/QA、candidate側HANDOVER拒否、completion no-ff第2親、optional Wiki、未検証差し替え拒否、abort後の品質証跡bytes/ステージ復元、lock/validation/commit各一回、hook digest一致/不一致/通常commitを確認する。
- checker/process tests: Git導出、reviewer独立監査、最小QA証跡、focused/full rerun選択、estimate/Wiki/CF/形式値/model mismatchが不要な停止にならないこと、identity/boundary違反が停止することを確認する。
- scope test: `ALLOW_MERGE=1`のexact no-ff candidate merge受理と未指定direct product path拒否を確認する。
- 最終ゲートは`make test-process`、`make task-check TASK=TASK-0037`、`make check`、`git diff --check`。本計画作成時は維持チェック以外を実行しない。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 最小既定と削除優先の方針を確認した。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] 見積が参考値であり、完了gateではないことを確認した。
- [x] DEV開始を承認した。
