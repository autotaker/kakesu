---
task_id: "TASK-0040"
change_class: product
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T05:24:39Z"
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0040 QA PLAN

## 方針

この計画は DEV 開始前に `TASK.md` の `planning input packet` だけから作成した。QA は HANDOVER に固定された同一 `candidate_commit` に対し、`internal/capability` の source/test audit と `cd tools/dev-agent-harness && GOCACHE=/tmp/task-0040-qa-gocache go test -race ./internal/capability` の一回だけの focused rerun を行う。DEV が candidate で実行した harness `make check`/`make distcheck` と root `make check` は candidate-bound 証跡を監査し、QA は再実行しない。

実 Credential、network、TLS、proxy、永続化、restart、複数 process、配置、cleanup は対象外である。`live-e2e` は割り当てず、それらの PASS を主張しない。ケース証跡はケース ID、HANDOVER の candidate、command、result のみとし、artifact digest、cache/exit 形式、テスト名への固定 mapping、追加 check を要求しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1, AC-2 | focused race test と source/test audit で、Rules の canonical validation、nil/zero deny、IssueSpec の正規 subject/workspace/non-root UID/provider scope/TTL/uses、opaque `cap_` handle、digest-only entry、entropy failure・bounded collision と partial entry 不残存を確認する。 | `focused-rerun` / memory 内の Rules/Issue fixture は hermetic・deterministic・上限付きである。 |
| QA-002 | AC-3 | GitHub/OpenAI の canonical scope allow と、subject/workspace/provider/repository/operation/host/handle の完全一致、prefix/suffix・大小文字・未知 handle を含む代表 deny、mismatch が uses を消費しないことを focused race test と source/test audit で確認する。 | `focused-rerun` / scope と Request の pure fixture で allow/deny 境界を完全に再現できる。 |
| QA-003 | AC-4 | expiry 境界、uses の一回だけの消費と最後の成功後の削除、Revoke、単調 epoch、epoch 更新後の無効化、同一 1-use handle の並行 Consume で exactly one success と race-free を focused race test と source/test audit で確認する。 | `focused-rerun` / clock/entropy の package 内 test dependency と bounded concurrency fixture で atomic lifecycle を再現できる。 |
| QA-004 | AC-5 | fixed error の non-leak、raw handle/Credential/Request slice の非保持・非変更、crypto entropy と UTC clock だけを production constructor が使うこと、file/environment/process/network/DNS/TLS 非使用、README の restart fail-safe と非永続/非通信境界を focused race test と source/test audit で確認する。 | `focused-rerun` / pure in-memory boundary と README scope は外部環境なしに確認できる。 |
| QA-005 | AC-6 | HANDOVER の `candidate_commit`、candidate diff、DEV の focused test、harness check/distcheck、root check の command/result を監査する。差分が capability package と harness README に限定され、1,200行以下で、tests が代表 issue/consume/deny/revoke/epoch/expiry/collision/entropy/non-leak/不変性/並行性を検出し、削除・期待値緩和がないことを確認する。 | `evidence-review` / candidate 束縛、scope、DEV の必須回帰実行と test の失敗検出能力は独立監査が必要である。 |

## 実行手順と期待結果

1. HANDOVER から `candidate_commit` を取得し、対象をその candidate に固定する。candidate が一意に読めない、HANDOVER が candidate 側で変更されている、又は candidate-bound 証跡が不足する場合は PASS にしない。
2. QA-001〜QA-004 は candidate で `cd tools/dev-agent-harness && GOCACHE=/tmp/task-0040-qa-gocache go test -race ./internal/capability` を一度だけ実行する。source/test audit と合わせて各観測対象を満たし、command が PASS することを期待する。package 又は十分な pure API boundary test が存在しない場合は `qa_plan_defect` 又は `requirement_gap` として Main に返し、広域 rerun で代替しない。
3. QA-005 は candidate diff、HANDOVER、DEV の command/result、QA-001〜004 の結果を監査する。許可外変更、1,200行超過、candidate 不一致、test の削除/弱体化、必須 DEV command の未実施/non-pass、又は secret があれば PASS にしない。`make task-check TASK=TASK-0040` は completion gate/merge 後の Main 所有であり、candidate QA には要求しない。

## 境界・異常・回帰

- handle は実 Credential 又は self-contained token ではなく、random reference の digest-only registry key でなければならない。scope の部分一致、mismatch 時の uses 消費、最後の成功/expiry/revoke/epoch 後の再利用、race は回帰として扱う。
- 高リスク信号、candidate/証跡不一致、scope 不明、tests の削除・弱体化、又は代表 negative evidence の欠落があれば、QA-005 の `evidence-review` を PASS にしない。focused test の green 結果で置き換えない。
- 初期 FAIL 分類は、candidate が AC と異なる場合は `implementation_defect`、Task packet の矛盾/不足は `requirement_gap`、本計画又は fixture の誤りは `qa_plan_defect`、実行不能な toolchain は `environment_issue`、既存 pure boundary の破壊は `regression` とする。最終分類は Main が証拠に基づき決定し、DEV fault と仮定しない。
- Credential/network/TLS/persistence の実証は対象外である。後続 Task がそれらを接続する場合は、承認済み環境と安全な cleanup を伴う別 `live-e2e` を計画する。

## 実装後の再確認

- [ ] fixed candidate の実装差分と REVIEW と独立の QA 結果を確認した。
- [ ] QA-001〜QA-004 の focused race test を candidate で一度だけ実行し、source/test audit と合わせて確認した。
- [ ] QA-005 の candidate-bound DEV 証跡と scope を監査し、期待結果または範囲を変更した場合は Main の承認を得た。
