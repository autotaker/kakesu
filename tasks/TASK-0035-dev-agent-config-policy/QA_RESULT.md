---
task_id: "TASK-0035"
status: passed
qa_agent: "qa-agent-terra-medium"
tested_commit: "5e5d29e8250d8b2999d2cf6e51e748b7f866b016"
candidate_commit: "5e5d29e8250d8b2999d2cf6e51e748b7f866b016"
candidate_tree: "cc3ea4ba4b6f99da68e73e370faddc3c1bad2aa1"
managed_path_digest: "a62367397b23bb54347ff9f04951b7a11e2eb2a97e1385c51b9a2e2aef395c40"
bootstrap_evidence_commit: ""
bootstrap_evidence_digest: ""
merge_tree: ""
decision: pass
tested_at: "2026-08-01T10:15:00+10:00"
---

# TASK-0035 QA RESULT

## 対象と環境

- candidate commit/treeとHANDOVERのmanaged digestは指定値と一致した。開始/終了時のcandidate worktreeはclean。QA PLAN revision 3を使用し、REVIEW_RESULT又はReviewerのPASSを開始条件・入力にせず実施した。
- 環境はDarwin arm64、Go `go1.26.5`、candidate worktree `worktrees/TASK-0035-dev-agent-config-policy/tools/dev-agent-harness`。`GOCACHE`/`GOMODCACHE`は`/private/tmp/task-0035-qa-*`、target testは`-count=1`、fixture/mutationはcandidate外の`/private/tmp`で作成・cleanupした。network、sudo、実`/etc`、service、実userは使用/変更していない。
- Main取得済み`make check` digest `c09c98dc056d1788fb622f303557cfe2642ac0d8d1762a7a44667a7503234ef5`、`make distcheck` digest `319226754bcd31e72083ca53681b351d6820f0b4df78d13b8f1e88136b46967a`、DEV diff digest `8bc852a6894cbe2077e083f2541726308d5d65b0d5fd527555e03685d0c4f9ed`をcandidate/treeと照合して再利用した。

## 結果

| ケースID | モード | 結果 | 証跡 | 未実施/blocked理由 |
|---|---|---|---|---|
| QA-035-01 | `focused-rerun` | pass | 自作0600 valid JSONをtemporary setup binaryへ渡しexit 0。stdoutは`config version=1 network.default=deny validated`、stderr 0 byte。path/user/credential sentinelは両streamに0件。fixture stream aggregate SHA-256 `2413d7b0627d4cc895974695b3565ddf2bf5e911116b04fd21172027240caf50`。 | なし |
| QA-035-02 | `focused-rerun` | pass | `go test -count=1 ./internal/config ./internal/command` exit 0。独自CLI fixtureでduplicate、unknown `allowlist`、version+trailingをexit 1、stdout空、入力sentinel非漏洩で確認（aggregate `2413d7b0627d4cc895974695b3565ddf2bf5e911116b04fd21172027240caf50`）。testはtop/nested unknown、duplicate、version、trailingをcase別に網羅する。 | なし |
| QA-035-03 | `focused-rerun` | pass | uncached target testで空/relative/nonclean path、重複/invalid user、deny以外network、V1不存在の`allowlist`をClassSemantic/ClassUnknownとして拒否。`allowlist:[]`の独自CLIもexit 1・非漏洩。field省略はunknown allowlistを許可へ変換しない。 | なし |
| QA-035-04 | `focused-rerun` | pass | 独自CLI fixtureでsymlink、FIFO、0660、0606、64KiB超を各exit 1、stdout空、file-policy診断かつsentinel非漏洩（aggregate `2413d7b0627d4cc895974695b3565ddf2bf5e911116b04fd21172027240caf50`、0606専用 `a276c35c8858206f2c88d4db5d9d87fd0c7b77eb69b10e6715529c1c824f74d1`）。uncached testはdirectoryも拒否する。sourceを独立監査し、O_NOFOLLOW/O_NONBLOCKでopen済みFDをread前後`f.Stat`しsamefile/type/mode/size/lengthを再検査するため、path置換raceで再解決しない。 | なし |
| QA-035-05 | `focused-rerun` | pass | category testはunknown/duplicate/version/trailing/path/user/network/allowlist/file-type/permission/sizeを含む。candidate外copyでunknown-field拒否をaccept側へ最小mutationし、`go -C copy test -count=1 -run TestParseRejectsStrictAndSemanticCases/(unknown|allowlist)`はexit 1、両subtest失敗（output digest `a5041df47109041356eca3838f8d7d8933b24ac28a2eaabe880cdbc06af690be`）。candidateは未変更。diffにtest削除/弱体化なし。 | なし |
| QA-035-06 | `focused-rerun` | pass | temporary buildした6 binaryは各`--version` exit 0、通常起動は全てexit 78かつ`refusing to start`をstderrへ出した。setup helpはREF-2 contractどおり。aggregate digest `28743f9bb10ebf5bdb38766b57e57bd8771559638beddd6ca61d8169bacd0a37`。 | なし |
| QA-035-07 | `focused-rerun` | pass | Mainのcandidate-bound `make check`/`make distcheck` PASS証跡と明示configure/install example検証を照合した。configure fixtureは`--prefix=/usr/local --sysconfdir=/etc --localstatedir=/var --runstatedir=/run`を明示し、temporary DESTDIRの展開済みexampleをinstall済みsetupでexit 0検証している。QAのuncached config/command rerunもpass。 | なし |
| QA-035-08 | `evidence-review` | pass | base `5d8ecf1`との差分はREADME 6、command 18+40、config 262+116行（計442、実装+test 436）。差分digestはHANDOVERと一致。許可pathだけ、go.mod差分/外部moduleなし。`git diff --check` exit 0。socket/network/process/credential/persistence/serviceの新実装は検出せず、FD read-only policyのみ。700行は目安で、機能/negative testを削らない436行はAC-8の上限・新boundary分割条件に抵触しない。 | なし |

## 発見事項

| ID | 分類 | 影響 | 差し戻し候補 | 内容 |
|---|---|---|---|---|
| - | - | - | - | なし |

## 未実施・Main判断

- blocked/未実施はない。negative mutationはcandidate外copyだけを変更しcleanup済みである。
- `qa_carry_forward`は`not-applicable`。指定candidateを直接評価した。設定/parser/file policy/CLI/installに関わる候補変更はcarry-forward禁止であり、変更時は影響ケースを再実行する。
- `merge_tree`はマージ前のため未設定。Mainはmerge後にcandidate treeとの同一性を確認する。本Taskにlive-e2eケースはない。

## 結論

`pass`。QA-035-01〜08は指定candidate commit/treeへ束縛してPASSした。FAIL、blocked、要件gapはない。
