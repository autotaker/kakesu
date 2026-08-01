---
task_id: "TASK-0039"
change_class: product
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T04:48:37Z"
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0039 QA PLAN

## 方針

この計画は DEV 開始前に `TASK.md` の `planning input packet` だけから作成した。QA は HANDOVER に固定された同一 `candidate_commit` を対象に、`internal/egresspolicy` の table-driven Go tests を candidate で一度だけ focused rerun する。DEV が candidate で実行した harness `make check`、`make distcheck`、root `make check` は証跡を監査し、QA は重複実行しない。

実 network、TLS、Credential、proxy/HTTP server、配置、restart、rollback、cleanup は Task の対象外であり、`live-e2e` を割り当てない。これらについて PASS を主張しない。QA 結果のケース証跡はケース ID、HANDOVER の candidate、command、result のみとし、cache/exit 詳細、artifact digest、version/mode、estimate 算術、全 Task Wiki receipt、CF、又は新しい形式の check を要求しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | Rules の有効作成と、空集合・重複・非 canonical identifier・無効上限の固定 Rules error、caller が Rules slice を変更した後の不変性を、focused Go test と source/test audit で確認する。 | `focused-rerun` / pure Rules fixture は hermetic・deterministic・上限付きである。 |
| QA-002 | AC-2 | GitHub の canonical allow（許可 repository の `GET`/`HEAD`）と、method/host/port/path/query/userinfo/fragment/encoding/dot/empty segment/allowlist 外/URL 上限の代表 deny を、focused Go test と source/test audit で確認する。 | `focused-rerun` / URL と Rules だけで完全に再現できる pure decision である。 |
| QA-003 | AC-3, AC-4 | OpenAI の canonical strict allow と、surface/content type/body 上限、malformed/trailing/duplicate/unknown JSON、model/input/store/stream/max output tokens/instructions の代表 deny を、focused Go test と source/test audit で確認する。 | `focused-rerun` / bounded JSON body と Rules だけで strict parser 境界を決定的に再現できる。 |
| QA-004 | AC-5 | nil/zero policy と invalid request の default deny、provider 固定 allow decision と単一 fixed deny error、入力由来値・parser/OS error の非漏洩、Request body/Rules/入力 slice の非保持・非変更、file/environment/process/network/DNS/TLS/clock/random 非使用を、focused Go test と candidate source/test audit で確認する。 | `focused-rerun` / pure package の fail-closed と不変性は隔離 fixture と静的監査で確認できる。 |
| QA-005 | AC-6 | HANDOVER の `candidate_commit` と candidate diff を突合し、DEV の `go test`、harness `make check`、`make distcheck`、root `make check` の candidate-bound command/result を監査する。許可 path が `internal/egresspolicy/` と harness README に限られ、tests が代表 allow/deny 境界を実際に検出し、削除・期待値緩和がないことを確認する。 | `evidence-review` / candidate 束縛、scope、DEV の必須回帰実行と test の失敗検出能力は独立監査が必要であり、root check の重複実行は不要である。 |

## 実行手順と期待結果

1. HANDOVER から `candidate_commit` を取得し、以後の対象をその candidate に固定する。candidate が一意に読めない、HANDOVER が candidate 側で変更されている、又は候補差分が固定できない場合は PASS にしない。
2. QA-001〜QA-004 は candidate で `cd tools/dev-agent-harness && go test ./internal/egresspolicy` を一度だけ実行する。source/test audit と合わせて上表の観測対象が満たされ、command が PASS することを期待する。package 又は十分な pure API boundary test が存在しない場合は `qa_plan_defect` 又は `requirement_gap` として Main に返し、別の広域 rerun で代替しない。
3. QA-005 は candidate diff、HANDOVER、DEV の command/result、及び QA-001〜004 の結果を監査する。許可外変更、candidate 不一致、テスト削除/弱体化、必須 DEV command の未実施/non-pass、又は secret があれば PASS にしない。`make task-check TASK=TASK-0039` は completion gate/merge 後の Main 所有であり、candidate QA には要求しない。

## 境界・異常・回帰

- 全 deny は provider 固有の詳細、URL、repository、model、body、Credential らしい値、又は parser/OS error を含まない同一固定 deny error でなければならない。allowlist の prefix/suffix 照合、normalization による曖昧入力の受理、OpenAI unknown field の許容は回帰として扱う。
- 高リスク信号、candidate/証跡不一致、scope 不明、テスト削除・弱体化、又は negative test がない場合、QA-005 の `evidence-review` を PASS にしない。focused test の green 結果で置き換えない。
- 初期 FAIL 分類は、candidate が AC と異なる場合は `implementation_defect`、Task packet の矛盾/不足は `requirement_gap`、本計画又は test 対応の誤りは `qa_plan_defect`、実行不能な toolchain は `environment_issue`、既存 pure boundary の破壊は `regression` とする。最終分類は Main が証拠に基づき決定し、DEV fault と仮定しない。
- network/TLS/Credential の実証は対象外である。後続 Task がそれらを接続する場合は、承認済み環境と安全な cleanup を伴う別 `live-e2e` を計画する。

## 実装後の再確認

- [ ] fixed candidate の実装差分と REVIEW と独立の QA 結果を確認した。
- [ ] QA-001〜QA-004 の focused Go command を candidate で一度だけ実行し、source/test audit と合わせて確認した。
- [ ] QA-005 の candidate-bound DEV 証跡と scope を監査し、期待結果または範囲を変更した場合は Main の承認を得た。
