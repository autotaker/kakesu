---
task_id: "TASK-0035"
status: passed
qa_agent: "qa-agent-terra-medium"
tested_commit: "873993eea4e37cf05460bf08270c54789315ee3c"
candidate_commit: "6b5d3495a0f61bd0a1b134926ef932dd65a5000b"
candidate_tree: "84b53854c139b23d992026175b8f979ae71d4df2"
managed_path_digest: "66f2d043bf7acc1a7801233e553f9bcf6a45fea4e164450866e180fba8ad93d9"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
merge_tree: "1d37a775501bda244b642f803dd47eb5cadec0d8"
decision: pass
tested_at: "2026-08-01T10:26:47+10:00"
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
- Main選択は`qa_carry_forward`。初回QA PASS candidate `5e5d29e8250d8b2999d2cf6e51e748b7f866b016` / tree `cc3ea4ba4b6f99da68e73e370faddc3c1bad2aa1`から、最終candidate `6b5d3495a0f61bd0a1b134926ef932dd65a5000b` / tree `84b53854c139b23d992026175b8f979ae71d4df2`へ結果を引き継ぐ。
- `CF-1`: complete。旧QA-035-01〜08 PASSは旧candidate commit/tree/digestへ束縛済み。
- `CF-2`: complete。旧新差分は`tools/dev-agent-harness/README.md`の4 add/4 delだけで、SHA-256 `562579d4c621dd4f28c499fcd6a3cd3bc6c6c3e041f8853a04c44c10676c0f47`。
- `CF-3`: complete。変更は実行されない用語表記だけで、命令の意味、製品挙動、runtime、test、Schema、設定、依存、生成物、外部契約、AC、QA_PLANを変更しない。
- `CF-4`: complete。影響QAケース集合は`[]`。
- `CF-5`: complete。独立Reviewerは最終candidateのharness `make check`、candidate README textlint、旧新diff検査をPASSし、挙動・test・安全契約への影響なしを確認した。
- `CF-6`: complete。QA FAIL、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/lifecycle/persistence/error/fail-closed、test削除/弱体化、影響不明、binding不一致はいずれも偽。
- `CF-7`: complete。Mainが旧新commit/tree、全差分digest、空の影響集合、Reviewer証拠、docs lint起因の表記訂正である理由を本sectionへ記録した。
- 最終no-ff merge `873993eea4e37cf05460bf08270c54789315ee3c`の第2親は最終candidate `6b5d3495a0f61bd0a1b134926ef932dd65a5000b`である。main管理証跡を含むmerge treeは`1d37a775501bda244b642f803dd47eb5cadec0d8`で、merge scope checkerは製品差分がHANDOVERのmanaged digestと一致することをPASSした。
- merge後の`PYTEST_ADDOPTS=--ignore=worktrees make check`はexit 0。除外対象は別worktreeの重複pytest収集だけで、mainの全product/process/docs検査を実行した。本Taskにlive-e2eケースはない。

## 結論

`pass`。QA-035-01〜08は初回candidateへ束縛してPASSし、閉じたCF-1〜CF-7によりdocs-only最終candidateへ引き継いだ。FAIL、blocked、要件gapはない。
