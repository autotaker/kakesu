---
task_id: "TASK-0070"
change_class: "product"
status: "approved"
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T06:04:03Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0070 QA PLAN

## 方針と固定candidate

この計画の受け入れ根拠は TASK の `Planning input packet` に限る。QA は Main が固定した単一の `candidate_commit` を checkout した状態で、次の一回だけの bounded focused rerun を独立に実施する。テストは hermetic な package test に閉じ、network、Git subprocess、clock、randomness、永続 state を必要としない。

```sh
cd tools/dev-agent-harness && go test -race ./internal/approvalmanifest
```

実装は、下記の表で指定する test matrix（又は同等以上にケースを一対一対応させる test 名）を candidate に含める。QA はその matrix が入力を固定し、下記の期待結果を実際に検出できること、テストが被験 API の digest/canonical encoder を oracle として再利用していないことも差分・テスト証跡から確認する。一時的なソケット、ネットワーク、外部 service は使わないため `live-e2e` は不要である。

## 受け入れ条件との対応

| ケース ID | AC-ID | focused-rerun での独立確認 | 期待結果 | モード / 理由 |
|---|---|---|---|---|
| QA-001 | AC-1 | 必須 field、identity/repository/remote 組、時刻、文字種、各上限の表駆動負例 | 正常値だけが構築され、各負例は固定 category と field だけで拒否される。生成・現在時刻・TTL/policy 判定を求めない。 | `focused-rerun` / pure validation は deterministic である。 |
| QA-002 | AC-2 | 1件、32件、33件、重複、branch subset 外、object ID 表記、全 sentinel/flag transition、no-op、順序保持の表駆動検査 | 正当な create/update/force/delete だけが通り、ref は入力順のまま、他は拒否される。 | `focused-rerun` / 境界と遷移は固定 fixture で網羅できる。 |
| QA-003 | AC-3 | package digest API/encoder を使わず、固定の valid proposal から期待 canonical payload bytes を明示 fixture として組み立て、標準 SHA-256 と固定 domain prefix で expected digest を計算する oracle | 公開 bytes と `sha256:<lowercase-hex>` が oracle と一致する。同一入力は bytes/digest が同一で、manifest の各束縛 scalar field と各 ref update の各 field、および ref 配列順を一つずつ変えると digest が変化する。digest を任意入力できない。 | `focused-rerun` / security property を独立 oracle で再現できる。 |
| QA-004 | AC-4 | encoder 出力の parse/encode byte equality、unknown/duplicate/missing key、key order、whitespace、escape、number/time/digest spelling、trailing bytes、digest tamper の table | 唯一の canonical bytes のみ通り、parse→encode は byte-identical。constructor と parser に同一 validation 負例が適用される。 | `focused-rerun` / parser tamper cases は hermetic である。 |
| QA-005 | AC-5 | constructor input slice/raw bytes、parser input、updates getter、encoding getterを取得後に mutation する alias-regression と、全負例 error の文字列/公開 diagnostic を値候補で走査する検査 | 内部値、bytes、digest、後続結果は変化しない。error は category と field/index 以外の入力由来値を漏らさず、過大/不正入力で panic しない。 | `focused-rerun` / memory alias と情報漏えいは unit/race で再現できる。 |
| QA-006 | AC-6 | 上記 command の `-race` 成功、candidate diff の許可3 path・変更行数目安・新規 dependency/I/O 等の非混入を確認し、DEV が candidate で実施した `make check`、`make distcheck`、`git diff --check` の command/exit status を candidate-bound evidence として監査する | race は PASS。scope は3許可 path内、約800〜1,100 changed lines目安で、外部依存・I/O・network・clock・randomness・Git subprocess・state persistence がない。3全体検査は成功証跡がある。 | `focused-rerun`（race/package）+ `evidence-review`（全体検査）/ QA が全体検査を重複して実行せず、candidate-bound 実行記録を独立監査する。 |

## focused test matrix の最低要件

- transition は少なくとも create、通常 update、明示 force、delete の正例と、zero→zero、非zero同値、flag/sentinel 不整合の負例を分離する。
- digest sensitivity は request、agent/workspace identity、repository、remote、policy/revocation、created/expires、および各 ref の ref/old/new/force/delete（該当表現）を一箇所ずつ変え、順序だけを入替える。全 field/ref order に対する変更検出を表で明示する。
- tamper は JSON として妥当に読める非canonical表現も含め、形式検査だけの通過を禁止する。digest tamper は正しい形式だが異なる値を使う。
- non-leak probe は identity、repository、remote、ref、object ID、digest、raw/canonical bytes の代表値を error text/diagnostic に含めないことを確認する。入力値を含む assertion failure を error contract の PASS 根拠にしない。
- 過大入力は各 bounded field、ref 数、raw encoding を対象にし、panic/無制限 allocation を起こさず固定分類で失敗することを確認する。

## 外部作用と後続境界

candidate の import と差分を確認し、network client、filesystem/process 操作、Git invocation、環境 clock/random source、永続 store を導入していないことを検査する。これは pure value boundary のため live E2E は不要であり、live E2E の PASS で置換もしない。

この QA は manifest の妥当性・canonical bytes・digest だけを対象にする。push の実行又は許可、Git wire/pkt-line、remote の old SHA 観測、Approval state、Passkey/WebAuthn、session、grant の署名/消費/reconciliation、実 credential は検証しない。manifest が valid でも approved/granted を意味しない。この境界外を candidate が実装又は挙動変更していた場合は scope FAIL として Main へ返し、focused-rerun の PASS で受け入れない。

## 判定と失敗分類

QA は同一 candidate の diff、focused-rerun 出力、全体検査の candidate-bound evidence を記録する。失敗は DEV fault と推定せず、少なくとも次へ分類する：実装/テスト不一致、テスト弱化・oracle 非独立、candidate/証跡取り違え、環境・harness、計画/受け入れ条件の曖昧さ、scope/統制逸脱。再現不能又は evidence 欠落は PASS にしない。

## 実装後の再確認

- [ ] Main が指定した `candidate_commit` と diff を確認した。
- [ ] focused-rerun を一回実行し、全ケース ID と結果を candidate に紐付けた。
- [ ] independent oracle、全 field/ref-order sensitivity、tamper、transition、alias/non-leak、過大入力、外部作用不在、scope/line budget を確認した。
- [ ] `make check`、`make distcheck`、`git diff --check` の candidate-bound evidence を監査した。
- [ ] 実行しなかった境界（push/Passkey/state）と live-E2E 不要理由を結果に記録した。

## 未決事項

- なし。候補実装に test matrix 相当のケースまたは独立oracleが存在しない場合は、QA 実施時の FAIL として扱う。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | QA Agent (Terra/medium) | Planning input packetのみから独立QA計画を作成 | `main-agent-sol-high` |
