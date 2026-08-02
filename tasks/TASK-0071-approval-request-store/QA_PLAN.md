---
task_id: "TASK-0071"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T06:51:59Z"
revision: 1
implementation_reviewed_at: "2026-08-02T07:39:03Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0071 QA PLAN

## 方針

TASK.md の Planning input packet と AC を唯一の期待値とする。candidate commit を固定後、`internal/approvalstate` のテストを `-race` で一回だけ実行する bounded focused suite を主証跡とする。suite は一時 directory、決定的 test clock、persistence/failure seam と有限数 goroutine/操作数だけを使い、state directory を作成・chown せず、外部 service、実 credential、network filesystem を必要としない。

各 mutation は、成功時の memory/disk 再 open 一致、失敗時の generation/records 非部分更新、エラーの固定 class と非漏えいを観測する。失敗注入、clock、lock と filesystem の OS 差は fixture 内に閉じ込める。candidate の root `make check`、harness `make check`、`make distcheck`、`git diff --check` は DEV が同一 candidate で提出した証跡を独立監査するだけであり、QA は重複実行しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | focused suite で absolute clean existing `0700` root の Open/Close、同 root 二重 Open、relative/traversal、wrong mode/type、state/lock symlink、Close 二回を検査する。現在 OS で実 lock を競合 open し、OS 非対応実装は supported OS と誤認せず固定 unsupported error を検査する。Linux/Darwin のうち非実行側は各 OS build 対象と platform-specific lock source の candidate evidence を監査する。 | `focused-rerun` / owner-only directory と単一 writer は hermetic な temp directory と非 block lock で再現できる。実行不能 OS は live 代替をせず compile/evidence として未実行理由を残す。 |
| QA-002 | AC-2 | same suite で TASK-0070 parser に通る canonical manifest のみを使い、policy/epoch、trusted clock 境界、TTL（0/上限/超過）、record 上限、request ID 未使用を Create 前提として表形式に網羅する。非canonical/parse 不可、mismatch、expired/future created、duplicate ID（同一 bytes を含む）を拒否し、caller が digest/state/time/path を注入できない API であることを compile-level API usage と結果で確認する。 | `focused-rerun` / deterministic manifest fixture と test-only clock により全境界を一回の bounded suite で再現できる。 |
| QA-003 | AC-3 | same suite で digest 一致の pending から approved/denied/cancelled/expired、および approved から stale/expired の全許可遷移を各一回確認する。digest/policy/epoch mismatch、terminal 再遷移、approved 以外の stale を拒否する。expiry 時刻ちょうど、decision と expiry の競合、Get と ExpireDue で active record が approved 扱いにならないこと、clock rollback の安全側拒否を確認する。decision actor の保存と、store が Passkey 検証を行わない境界も API/依存差分から監査する。 | `focused-rerun` / fixed clock と bounded concurrent operation で expiry 優先と原子的遷移を確定的に検出できる。 |
| QA-004 | AC-4 | same suite で複数 request の request-ID 順 canonical snapshot、generation/version、restart re-open を検査する。partial/trailing/unknown/duplicate/noncanonical/oversize snapshot、record 整合不良、残存 temp、permission/symlink を fail-closed とする。write、file fsync、rename、directory fsync の各失敗注入について、rename 前は旧 disk/memory 維持、rename 後の不確実は poison 後に全操作が拒否され re-open/reconciliation まで回復を推測しないことを確認する。 | `focused-rerun` / persistence seam で filesystem failure points と破損 bytes を有限 fixture として再現できる。 |
| QA-005 | AC-5 | same suite を `go test -race` で実行し、Create/Get/decision/ExpireDue/Close の同時開始、Close と各操作の競合、nil/closed/poisoned を有限回繰り返す。public record、encoding、list getter の返却値を mutation 後に再 Get して ownership 分離を確認し、固定 error class に root/request/actor/repository/ref/SHA/digest/bytes/lower error が含まれないことを sentinel 値で検査する。失敗 mutation 後の generation/record 不変、panic/deadlock/data race 不在を確認する。 | `focused-rerun` / race detector と timeout 付き有限 concurrency は hermetic・deterministic に近く、AC の並行安全性を直接検査できる。 |
| QA-006 | AC-6 | candidate diff と module/dependency metadata を監査し、許可 5 paths、stdlib と既存 `approvalmanifest` のみ、HTTP/credential/Git/external DB/config/deploy/generated/live state 不在、README の状態・durability・single-writer・authorization 境界・後続 Task 記載を確認する。DEV 提出の root/harness check、distcheck、diff check の command、candidate SHA、exit status を監査する。 | `evidence-review` / root/harness/distcheck は同一 candidate の DEV 証跡を監査し、QA が重複実行しない。focused suite の PASS をそれらの PASS と取り違えない。 |

## focused suite の固定実施手順

candidate SHA を HANDOVER の `candidate_commit` と一致確認してから、対象 package の全テストを race detector、有効な timeout、失敗時の test name を出す設定で一回だけ実行する。実装で test target が定義されている場合はその target を優先し、そうでなければ package scoped `go test -race` を使う。実行前後に candidate SHA と対象 package diff を照合し、実行中の差分変化があれば結果を invalid として Main に返す。

テストは次の bounded fixture 群を一回の invocation に含める。ランダム入力や無制限 retry は使用せず、concurrency は開始 barrier、有限 goroutine、timeout で固定する。

- Open fixture: normal root と AC-1 の拒否対象を個別 temp root に作り、supported platform の lock 競合を一組だけ検査する。
- Create/transition fixture: 一つの canonical baseline と各違反を単一変数で変え、Create rules、全遷移、expiry 優先、rollback を検査する。
- durable fixture: 初回 create、各 successful mutation、restart、各 corruption と pre/post-rename failure を独立 store instance で検査する。
- safety fixture: poison、public-output mutation、error redaction、nil/closed、bounded mixed-operation concurrency/Close を検査する。

## 環境依存・live E2E の扱い

実 UID、実 `/var/lib` 配置と permission、VPS/systemd provision・restart・rollback、OS/process 境界で verified decision API を限定する接続、Passkey/WebAuthn、smartphone/push と実 push はこの Task の対象外であり `live-e2e` の PASS を要求しない。将来それらを導入する Task は、承認済み実環境、cleanup、restart/rollback 手順を別途計画して live-e2e とする。Linux/Darwin の実行不能側も、この Task では compile/evidence のみで、実機 PASS を主張しない。

## 判定・失敗分類

focused suite FAIL は直ちに DEV fault としない。QA は test assertion と AC の対応、fixture の妥当性、candidate SHA、host/OS 差、race/timeout、failure seam の注入位置を確認し、(a) implementation defect、(b) test/fixture defect、(c) environment/platform limitation、(d) requirement/evidence ambiguity、(e) candidate/evidence mismatch に分類する。高リスク durability/poison/expiry/concurrency で証跡欠落または影響不明なら `evidence-review` PASS で補完せず FAIL/blocked とする。

## 実装後の再確認

- [ ] HANDOVER の `candidate_commit`、suite 実行時 SHA、提出済み check evidence が同一 candidate である。
- [ ] QA-001〜QA-005 を一回の bounded focused race suite で独立実行し、各 test/case の PASS/FAIL/未実行理由を記録した。
- [ ] QA-006 の scope、依存不在、README、candidate evidence を監査し、root/harness check/distcheck を再実行していない。
- [ ] Linux/Darwin の未実行側と live 対象外を PASS と誤記していない。
- [ ] 実装差分と独立レビュー結果を確認し、期待値または範囲変更時は Main 承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | QA | Planning input packet に基づく独立初版 | `main-agent-sol-high` |
