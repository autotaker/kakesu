---
task_id: "TASK-0038"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T03:36:53Z"
revision: 2
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0038 QA PLAN

## 方針

TASK本文だけを仕様として、同一candidate commitを独立に検証する。`provision.Build(config, target-root)`が返すcanonical bytesとの完全一致だけを受理条件とし、独立parser、JSON grammar matrix、又は別schemaをテスト・実装へ要求しない。Ubuntu/root/executorは対象外であり、`live-e2e`は不要である。

## 受け入れ条件との対応

| ケース ID | AC-ID | fixture / command | 期待結果 | 実施モード / 理由 |
|---|---|---|---|---|
| QA-001 | AC-1, AC-2 | 有効V1 configとabsolute/clean target rootから`provision.Build`が生成した安全なregular manifestをtemp dirに置き、CLIを実行する。 | exit 0、stdoutが厳密に`provision manifest version=1 actions=10 verified\n`、stderr空。canonical bytesと同一の入力だけを受理する。 | `focused-rerun` / hermetic unit/CLI fixtureで成功契約を再現できる。 |
| QA-002 | AC-2, AC-6 | QA-001のcanonical manifestに対し、代表的に1 byteを追加、変更、削除する。さらに同一manifestを異なる有効config又はtarget rootで検証する。 | 各入力はnon-zeroかつstdout空。JSONとして読めるかに関わらずcanonical bytes不一致を拒否する。 | `focused-rerun` / 最小差分のbyte mutationとconfig/root mismatchで完全一致境界を再現できる。 |
| QA-003 | AC-3, AC-6 | symlink、non-regular file、128 KiB超、group writable、world writableのfixtureを渡す。ちょうど128 KiBの安全なregular fileは、loaderのfile policyを通過した後にcanonical mismatchとなる境界fixtureとする。open済みFDをread中に小さくかつ決定的に切詰め又はchmodするtest seamを発火させる。 | unsafe fixtureおよび読取前後でsize/modeが変わるfixtureはnon-zeroかつstdout空。ちょうど128 KiBのfixtureはfile policy通過後、canonical mismatchとしてnon-zeroかつstdout空。検査・読取りはsingle FDであり、検査後のpath再openはない。 | `focused-rerun` / temp fixtureと明示seamでfile policyとTOCTOU拒否を決定的に検証できる。 |
| QA-004 | AC-4, AC-6 | 引数不足/余剰、設定不正、canonical mismatch、file policy違反、注入したread errorを実行する。path、user名、config/manifest本文、JSON値、OS error文を互いに異なるsentinelにする。 | 全失敗がnon-zeroかつstdout空。stderrは固定error classだけで、全sentinel及びOS error本文を含まない。 | `focused-rerun` / CLIの機械比較で失敗契約と非漏洩を検証できる。 |
| QA-005 | AC-5, AC-6 | QA-001〜004を実行前後にmanifest/config/target rootとtemp rootのbytes、mode、directory listingで比較する。test double又は監視seamでchild process、network、IPCを失敗させ、既存の他setup操作と5 binaryのfail-closed testも実行する。 | 成功・失敗のいずれも入力/target root/host stateを変更せず、process/network/IPC開始は0回。既存のfail-closed結果は不変。 | `focused-rerun` / read-only境界を外部環境なしに観測できる。 |
| QA-006 | AC-7, AC-8 | candidateで `cd tools/dev-agent-harness && ./configure && make check && make distcheck`、続けてrootの`make check`を独立実行する。candidate diffで`configure`、install surface、許可外path、executor/実OS/root権限の追加、手書き差分1,200行超過を確認する。 | 全commandがPASSし、`configure`とinstall surfaceは不変。許可外path、新security boundary、又は1,200行超過ならPASSにせずMainへ戻す。失敗はcandidate regression、baseline、環境、test flaw、仕様/QA不整合に分類し、DEV faultと即断しない。 | `focused-rerun` / 指定回帰commandとcandidate scopeを同一案に対して独立確認する。 |

## 実装後の再確認

- [ ] 各ケースのcandidate commit、command、fixture、結果又は未実施理由を記録した。
- [ ] QA-002〜005をDEV結果と独立に再実行し、失敗を原因別に分類した。
- [ ] candidate diffに独立parser/別schema、path再open、入力由来の診断、外部作用、生成済み`configure`又はinstall surfaceの変更がないことを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA Planner | 初版 | `superseded` |
| 2 | 2026-08-01 | qa-agent-terra-medium | 更新TASKに合わせ、canonical equality中心の6ケースへ縮約 | `main-agent-sol-high / 2026-08-01T03:36:53Z` |
