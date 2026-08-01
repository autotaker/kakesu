---
task_id: "TASK-0036"
change_class: "product"
status: draft
qa_agent: "qa-agent-terra-medium"
approved_by: ""
approved_at: ""
revision: 3
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0036 QA PLAN

## 独立性・固定条件

本計画は`TASK.md`とそのREF-1〜4だけから作成し、`PLAN.md`を入力にしない。評価対象はDEVが固定する同一の`candidate_commit`と`candidate_tree`であり、REF-4のbootstrap digest `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`を引継ぎ証跡として照合する。各実行はcase ID、commit/tree、コマンド、fixture、cache条件、exit、stdout/stderr digest、snapshot/テスト成果物 digest、未実施理由をcandidate-bound HANDOVERへ残す。不一致、証跡不足、テスト削除/弱体化、又は影響不明なら`evidence-review`をPASSにしない。

有効fixtureは、version 1・`network.default=deny`・相異なるagent/runtime/broker user、logical absoluteかつcleanなconfig/state/runtime pathを持つ最小regular config、空の一時`target-root`である。各focused rerunは独立temp directory、network/process/IPC seamのfake、byte snapshotを使い、cacheは無効又はcache key/状態を記録する。live host、sudo/PAM、systemd実行、install、executorは対象外なので`live-e2e`は0件である。

失敗は観測根拠に従い`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`に仮分類する。分類はDEV faultを前提にせず、最終判断はMainが行う。

## 受け入れ条件との対応

| ケースID | AC | 実行モード（理由） | fixture / 観測と期待 | FAIL候補 |
|---|---|---|---|---|
| QA-001 | AC-1,2 | `focused-rerun` — pure CLI、固定input/outputでhermeticかつdeterministic | 有効fixtureでCLIを最低2回実行。両方exit 0、stderr空、stdoutは末尾改行を含むcanonical JSONL **11行**だけ、SHA-256同一。header 1行は`kind=manifest`,`version=1`,`platform=ubuntu`,`default=deny`,`target_root`,`action_count=10`、後続は連番1〜10でuser→directory→service。各lineをJSON decodeし、余分なline/空白/fieldの順序揺れを失敗とする。 | implementation_defect / environment_issue |
| QA-002 | AC-3 | `focused-rerun` — in-memory manifest validationに閉じる | QA-001出力のaction 1〜3をexact-field assertion。configのagent/runtime/brokerを各roleへ一意対応、home `/nonexistent`、shell `/usr/sbin/nologin`、`locked=true`、`create_home=false`。重複user又はrole/name誤対応fixtureも拒否/出力不能であること。 | implementation_defect / requirement_gap |
| QA-003 | AC-4 | `focused-rerun` — path mappingはhost非依存の純粋関数として完全再現可能 | action 4〜7がconfig/state/runtime/auditのlogical path、root配下target path、4桁`0750`、owner/groupを持つことをexact assertion。config=`root:broker`、他=`broker:broker`、auditはstate配下。正常rootとlogical pathsの組合せでtarget containmentを再計算し、相対化・join重複を検出する。 | implementation_defect |
| QA-004 | AC-4,6 | `focused-rerun` — table-driven invalid root/path試験はhermetic | empty、relative、`.`/`..`、double separator等non-clean、NUL入りtarget-root、escapeを試すlogical path/configを個別実行。全てnon-zero、stdout空、入力path/config本文/userをstderrに含めず、target pathがroot外へ出ない。 | implementation_defect / qa_plan_defect |
| QA-005 | AC-5 | `focused-rerun` — fixed recordsを厳密比較できる | action 8〜10はbroker/egress/approvalの固定名、config broker user、`enabled=false`,`started=false`をexact assertion。agent/runtime userをservice userとするrecord、enabled/started true、追加serviceをnegative mutationで検出する。 | implementation_defect |
| QA-006 | AC-6 | `focused-rerun` — bounded invalid-argument/config harness | `--config`/`--target-root`の不足、余剰、順序違い、空値、invalid config（version/unknown/duplicate/semantic/file policy）をtable-driven実行。各々non-zero/stdout空、stderrが入力path・user値・config本文を漏らさない。正常configと同じ実行環境で確認する。 | implementation_defect / regression |
| QA-007 | AC-6 | `focused-rerun` — controlled failing writerで一回だけのwrite経路を観測できる | manifest encoder/writer seamへprefix+errorを返す一回のWrite失敗を注入。partial bytesは許容し、必須はnon-zero、retry/re-emitなし（call logとstdout digestで確認）、secret-like config値及び入力pathがstderrに無いこと。 | implementation_defect |
| QA-008 | AC-7 | `focused-rerun` — temp-root snapshotとinjected seamsで外部作用を直接検査できる | CLI前後のtarget-root tree（path/type/mode/owner/content digest）とhost sentinelを比較し不変。process executor、network dialer、IPC/systemd seamはpanic/counting fakeにし、起動/接続/call count=0をassert。read-only config open以外の作成/chmod/chownを検出したらFAIL。 | implementation_defect / environment_issue |
| QA-009 | AC-7 | `evidence-review` — 同一candidateの全binary fail-closed性と差分範囲は独立監査が必要 | setup以外の5 binariesとsetupの未許可operational invocationを確認し既存どおりrefusal/non-success、process/network/IPC非起動。candidate diffが許可3パスに閉じ、sysusers/tmpfiles/systemd/configure/build/install templatesに触れないこと、seam testsが実際にguardを検出することをレビューする。 | regression / implementation_defect / requirement_gap |
| QA-010 | AC-8 | `focused-rerun` — repository-defined deterministic checksをcandidate上で限定再実行 | `go test ./...`（harness cwd）、harness `make check`、root `PYTEST_ADDOPTS=--ignore=worktrees make check`、`make distcheck`、root docs lintを記録されたtoolchain/cache条件で実行しexit 0。テスト又はツール不在はPASSで代替せずenvironment_issueとして証跡化。 | environment_issue / regression / implementation_defect |
| QA-011 | AC-8 | `evidence-review` — mutationの有効性は単なるgreen runでは証明できない | order、disabled/stopped、target containment、writer errorの各guardを一つずつ負に変異し、対応testが失敗するDEV証跡を独立確認。mutationが実行不能なら理由・影響を記録してblocked。テスト削除、assertion緩和、mutation未検出はPASS不可。 | implementation_defect / qa_plan_defect |
| QA-012 | AC-9 | `evidence-review` — line/scope/security-boundary判定は差分監査 | hand-written implementation+testのSLOCを記録し700〜1,200目安からの逸脱を判定。1,200超、executor/OS変更、sudo/PAM/systemctl/sysusers/tmpfiles又はnew security boundaryの導入はMainへ即報告し、未承認ならblocked。許可外path変更も同様。 | requirement_gap / implementation_defect |

## 実施ゲートとマージ後

- QA-001〜008,010は同一candidate treeで独立に開始するfocused rerunである。QA-009,011,012のevidence-reviewは高リスク信号、candidate/tree/digest不一致、又は十分なnegative evidenceの欠落があればFAIL/blockedであり、greenテストで置換しない。
- QA-010のdocs lintはREADMEに変更がある場合に必須、変更がない場合もroot lintを実行して記録する。`make distcheck`は配布物に意図しないtemplate/生成物変更がないことを合わせて確認する。
- `qa_carry_forward`はMainのみがCF-1〜CF-7を全て証明した場合に選択できる。本Taskでは設定、error/fail-closed、path containment、lifecycle/IPC seam、test弱体化又は影響不明を含む変更にはcarry-forwardを使わず、影響ケースを再実行する。
- `merge_tree == candidate_tree`であり環境依存ケースが0件なら重複全面実行を省略できる。それ以外はcandidate bindingを失ったものとして該当ケースを再実行する。

## 未実施・blockedの規則

`live-e2e`は0件で、実Ubuntuへ作用しないというTASK境界を理由に必要としない。focused rerunのfixture、seam、mutation、toolchain、又はcandidate-bound成果物が欠ける場合は未実施理由と残余リスクを記録し、別モードのPASSで代替しない。TASKの観測値を変更する必要がある矛盾はQAが補正せず、`requirement_gap`としてMainへ即時報告する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 3 | 2026-08-01 | qa-agent-terra-medium | writer partial-write境界とbinary数をTASK境界へ訂正 | pending |
| 2 | 2026-08-01 | qa/Terra/medium | TASK-firstの独立ケース計画 | pending |
