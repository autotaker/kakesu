---
task_id: "TASK-0060"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T23:26:36Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0060 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。candidateの許可path内で、新Task template、safety contract checker、統制文書、process testを確認し、独立計画Reviewerを廃止してもMainの承認・分類・契約検査と、DEV後の独立REVIEW/QAが維持されることを評価する。

これは開発workflow toolingのproduct変更であり、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、依存、生成物、外部サービスは対象外である。live E2Eは割り当てない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | 新Taskの`PLAN.md` templateに`planning_reviewed_by`、`planning_review_decision`、`planning_reviewed_at`、独立計画Reviewerの必須指示がないことを確認する。既存の`approved_by`/`approved_at`によるMainのPLAN/QA_PLAN承認だけでDEV開始を表現できることを確認し、削除fieldの再導入、Main承認の削除、new version/receipt/checklist fieldをnegativeとして検出する。 | `focused-rerun` / template/process fixtureはhermetic・deterministic・boundedである。 |
| QA-002 | AC-2 | safety contract checkerがReviewer計画PASS・reviewer identity・review timestamp順序を要求しないことを確認する。同時にMainのPLAN/QA_PLAN承認、classification approval、Task-first QA_PLAN、宣言path/生成path、契約固有のscope/安全検査の欠落・矛盾・spoofを従来どおりFAILにするnegative fixtureを確認する。 | `focused-rerun` / temporary repository process fixtureで、廃止対象と残存fail-closed境界を完全に再現できる。 |
| QA-003 | AC-3 | `AGENTS.md`、Agent責務、development process、task managementが、Mainの軽量確認を意図・scope・受け入れ経路だけに限定し、PLAN技術内容の独立reviewを工程として要求しないことを確認する。PlannerのPLAN作成、QAのTASK-first QA_PLAN作成、Main承認を維持し、独立計画Reviewer必須化、Mainの範囲を超える新承認工程、role/model契約の変更をnegativeとして検出する。 | `focused-rerun` / source/process testは外部環境なしに契約文とroutingを照合できる。 |
| QA-004 | AC-4 | fixed candidateに対する独立実装REVIEW/QA、DEV/Reviewer/QA分離、candidate-bound check、no-ff completionがchecker/process testに残ることを確認する。Reviewer又はQAの欠落・identity不一致・candidate/HANDOVER不一致・no-ffでないcompletionがFAILし、計画Reviewerだけの欠落はDEV開始を妨げないことをnegative fixtureで確認する。 | `focused-rerun` / lifecycle fixtureで実装後品質gateと廃止した計画gateを決定的に区別できる。 |
| QA-005 | AC-5 | 既存Taskに残る`planning_review_*` fieldをcandidateが書換えず、checkerが互換入力として受理すること、新Task template/checkerから情報量のない必須fieldと専用FAIL fixtureだけが除去されることを確認する。許可path外、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、dependency、生成物の変更を検出する。DEVのroot `make check`と`git diff --check`、Reviewerによるcandidate/root check証跡をcommand/cwd/resultまで独立監査し、QAはroot full checkを重複実行しない。 | `evidence-review` / candidate diff、focused test本文とcandidate-bound DEV/Reviewer証跡で非対象不変と互換性を監査できる。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜004を次の一回だけ実行する。testはnew template、Main承認/分類/契約検査、old field互換、実装後REVIEW/QA/no-ffの少なくとも一つを壊すと失敗する必要がある。

```sh
node --test scripts/task/development-process.test.mjs
```

zero exitだけでは不十分であり、対象testの欠落、skip、弱体化、candidate不一致、またはrequired negative scenarioの欠落は該当ケースをPASSにしない。root `make check`はDEV evidenceとReviewerの独立監査に限り、QAは再実行しない。

## 境界・異常・回帰

- Mainの計画確認は意図・scope・受け入れ経路だけであり、独立計画Reviewer、専用PASS、時刻順序、new version/receipt/checklist fieldを再導入しない。
- TASK-firstのQA_PLAN作成とMain承認、PlannerのPLAN作成、classification approval、DEV profile、許可path、秘密境界を維持する。旧`planning_review_*`は既存Taskの互換入力であり、遡及削除・書換えしない。
- DEV後は独立REVIEW/QA、担当分離、同一candidate、candidate-bound check、no-ff completionを維持する。これらを計画軽量化の名目で弱める変更はFAILである。
- 許可path外、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、dependency、生成物、外部観測可能な製品挙動の変更はscope failureとして扱う。failureをDEV不具合と決めつけず、candidate、environment、requirement、証跡のいずれかに分類する。

## 実装後の再確認

- [ ] 同一candidateでQA-001〜005を独立に評価し、指定focused process testを一回だけ実行した。
- [ ] 新templateからの計画Reviewer field/指示不在、Main承認とclassification/safety検査、old field互換、実装後REVIEW/QA/no-ffのnegative failure-detectionを確認した。
- [ ] DEVのroot `make check`とdiff check、Reviewerのcandidate/root check証跡を監査し、QAはroot full checkを重複実行していない。
- [ ] live E2Eを計画せず、Kakesu/runtimeと`tools/dev-agent-harness` runtime不変を確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。計画Reviewer廃止、Main承認/契約検査維持、old field互換、実装後REVIEW/QA不変を定義。 | `approved` |
