---
task_id: "TASK-0036"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "標準libraryだけを使う純粋なmanifest組立てと既存setup CLIへの局所配線であり、OS executor、network、IPC、Credential、永続stateを追加せず、約810行の手書き実装・testをhermeticに検証できる候補であるため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:41:15+10:00"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T10:41:15+10:00"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
planned_implementation_files: 2
planned_implementation_lines: 340
estimate_points: 2
---

# TASK-0036 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `plan-provision --config PATH --target-root PATH`だけをsetupの許可操作にする。valid Configからheader 1 recordと固定10 action recordを構築し、struct固定fieldをcompact JSONで順に一行ずつ符号化したbyte列をstdoutへ単一writeする。map、時刻、host照会を使わない。 | `tools/dev-agent-harness/internal/provision/`、`tools/dev-agent-harness/internal/command/`、`tools/dev-agent-harness/README.md` | 1, 3, 5 | config/root/record validationまたはencode失敗はstdoutへ書く前に安全な分類だけをstderrへ出してnon-zeroにする。writer errorはnon-zeroで終了し、同一manifestのretry/re-emitをしない。 |
| AC-2 | canonical header型を`kind`, `version`, `platform`, `default`, `target_root`, `action_count`の順で固定する。canonical action共通型は`kind`, `sequence`, `action`を先頭に置き、user (1–3)、directory (4–7)、service (8–10) の各具体型を固定sliceへappendする。`kind=manifest/version=1/platform=ubuntu/default=deny/action_count=10`をvalidatorで確認する。 | `internal/provision/` とそのtest、`internal/command/` とそのtest | 1, 2, 3 | sequenceの欠番/重複、kind・version・platform・deny・countまたは群順序の不一致はinternal validation errorとして拒否し、出力しない。 |
| AC-3 | user actionはrole順`agent`, `runtime`, `broker`、設定の各user名を一回だけ使う。各recordの不変値を`home=/nonexistent`, `shell=/usr/sbin/nologin`, `locked=true`, `create_home=false`として型とvalidatorに埋め込む。 | `internal/provision/` とtest | 1, 2 | Configのstrict validationを再利用しつつ、role/nameの対応、重複、固定属性の逸脱をprovision層でもfail-closedにする。user作成、group変更、login操作は呼ばない。 |
| AC-4 | directory actionはlogical `config_dir`, `state_dir`, `runtime_dir`, `state_dir + /audit`の順に出す。directory recordは`logical_path`, `target_path`, `mode`, `owner`, `group`を固定fieldとして持ち、mode文字列は全件`0750`、configは`root:broker`、残りは`broker:broker`にする。auditのlogical pathがstate配下であることも検証する。 | `internal/provision/` とtest | 1, 2 | path空、NUL、非absolute/non-clean、auditのstate外、owner/group/mode逸脱は拒否する。ディレクトリ作成・stat・chmod/chownを行わない。 |
| AC-5 | service actionは`dev-agent-broker`, `dev-agent-egress`, `dev-agent-approval`の順にし、全recordへbroker user、`enabled=false`, `started=false`を固定する。Agent/Runtime userやenable/startを表すfield/valueを導入しない。 | `internal/provision/` とtest | 1, 2 | service名、user、disabled/stoppedのいずれかが逸脱したmanifestは出力せず、`systemctl`、exec、IPCを開始しない。 |
| AC-6 | root validationは空、NUL、relative、`filepath.Clean(root) != root`を拒否する。`MapTarget(root, logical)`はlogical先頭separatorを除く相対成分だけをjoinし、`filepath.Rel`で得る結果が`..`または`../`で始まらず、再clean後もroot配下であることを確認する。CLI引数は正確に`--config value --target-root value`の4 tokenだけ受理する。 | `internal/provision/`、`internal/command/` と各test | 1, 3 | path/config/argument errorはnon-zeroかつstdout空、stderrは固定診断（path/user/config本文なし）にする。writerの一回writeがshort/errorならnon-zeroで終了する。OS writerは一部byteを返してからerrorになり得るため、そのprefixを巻戻す/再送する保証は置かず、完全record事前validationと一回writeを原子化上限としてtest・READMEに明記する。 |
| AC-7 | provision packageはConfig値と文字列だけを入力にする純粋関数とし、CLI testはinjectしたwriterで観測する。temp target rootのtree snapshotを前後比較し、process/network/IPC seamはprovision/command packageに`os/exec`、net、socket、systemctl相当を持たない静的な依存境界として回帰する。既存共通`Run`の他5 binary/通常起動拒否を保持する。 | `internal/provision/*_test.go`、`internal/command/*_test.go` | 2, 3, 4 | snapshot差分、外部作用seam、他binaryのfail-closed回帰、またはtestが観測不能ならPASSにせずMainへgapを報告する。live OS検証、sudo/PAM、service実行は実施しない。 |
| AC-8 | valid fixtureでraw JSONLをbyte比較し、各lineをdecodeしてheader/action不変条件も確認する。negative mutationはorder、disabled/stopped、target containment、writer errorのguardを各々反転/除去して対応testがredになる候補証跡を残す。READMEにread-only usage、10 records、外部作用なし、executor境界とwriter上限を記載する。 | `internal/provision/*_test.go`、`internal/command/*_test.go`、`README.md` | 2, 4, 5 | `go test ./...`、harness `make check`、`make distcheck`、root docs lint/root checkのいずれかが失敗ならcandidate固定前にFAIL。README更新直後に`make lint-docs`、candidate前に指定root checkを行う。 |
| AC-9 | 標準libraryのみ。production約340行、test約470行、docs除外で計約810行に収める。新executor、OS/template/configure変更、1,200行超過見込みは実装を止めMainへ返す。 | 許可された3 pathのみ | 全工程 | `go.mod`、deploy template、configure/configure.ac/Makefile、systemd/sysusers/tmpfiles、host OSへの変更はscope breach。Main判断なしにTask分割やsecurity boundary追加をしない。 |

## 関連Wikiと判断

- Mainは完了後、manifest V1のrecord順序、target containment、全record事前validation、外部作用なしとexecutorとの境界をSemantic Wikiへ取り込む。本TaskのDEVはWikiやTask証跡を編集しない。
- REF-2のstrict Configを入力境界として再利用し、REF-3のsysusers/tmpfiles/systemd templateにある名前・所有・directory関係を読み取るが、それらのtemplateやconfigure契約は変更しない。REF-4のbootstrap bindingはcandidate固定以降のHANDOVER/REVIEW/QAへMainが継承する。

## 補足設計

### 代替案と不採用理由

- recordごとにwriterへ`json.Encoder.Encode`する方式: 後半recordのvalidation/encode errorで前半recordだけが見えるため不採用。全recordのvalidateとbyte列化を完了してから、一回だけwriterへ渡す。
- shell command列またはsystemd/sysusers/tmpfilesへの委譲: 現在の望ましい状態ではなく実行機構・OS権限を導入するため不採用。executorは後続Taskでmanifest consumerとして審査する。
- mapをmarshalするJSON: field順序をGoのmap挙動へ委ねるため不採用。header/action別structと固定sliceでcanonical field/orderを固定する。
- string prefixだけのtarget containment: `..`やseparator境界を誤判定し得るため不採用。absolute/clean/NUL検査、relative component join、`filepath.Rel`による再確認を併用する。

### 責務・境界・不変条件

- `internal/provision`がV1 manifest型、target mapping、固定desired-state組立て、全record validation、canonical JSONL byte列化を所有する。入力は`config.Config`とroot文字列だけで、filesystemのread/write、host user/group lookup、process起動、network/IPCをしない。
- `internal/command`はsetup subcommandの厳密な引数順、Config Load、provision呼出し、stdout/stderr/exit codeへ限定する。成功時stdoutはJSONLだけ、stderrは空。失敗時diagnosticは固定classだけで入力path/user/config本文を含めない。
- writer atomicityはapplication内の一回`Write`までが上限である。全recordを先に検査・serializeしてpartial *record*を避けるが、任意のOS/pipe writerがerror前に返したbyteの取消しは不可能であり、retry/re-emitもしない。
- `target_root`は実OS rootの代理となる文字列にすぎず、mapping/manifest生成はtargetをstatせず、directory/user/serviceを作成・変更・開始しない。Config loader以外のhost外部状態を観測しない。

### 移行・互換性

- 既存`check-config --config PATH`、`--version`、`--help`、setup以外5 binaryの通常起動拒否を保持する。追加する成功経路はsetupの新規サブコマンドだけである。
- manifest JSONLはversion 1を固定し、field追加、曖昧なdefault、future version fallbackを導入しない。consumer/executorは本Taskに含めず、actual Ubuntu executorがreadyになっても`Dependency-ready reconciliation`どおり別Taskへ送る。
- Configのlogical pathはhost pathを参照せずtarget root下の表示用target pathへ純粋に写像する。READMEはこのdry-runを権限付与またはprovision実行と誤解させない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `tools/dev-agent-harness/internal/provision/provision.go` | implementation | 230 | record struct、root/logical target mapping、固定V1 desired-state組立て、全record validation、canonical JSONL one-write payload。 |
| `tools/dev-agent-harness/internal/command/command.go` | implementation | 110 | `plan-provision`の厳密argument/Config/provision/writer/exit adapter。既存fail-closed面を保持。 |
| `tools/dev-agent-harness/internal/provision/provision_test.go` | test | 350 | raw JSONL determinism、header/10 action schema/order/attributes、mapping/containment、invalid root、pure no-mutation snapshot、negative mutation guard。 |
| `tools/dev-agent-harness/internal/command/command_test.go` | test | 120 | CLI stdout/stderr/exit、config/argument non-leak、writer short/error、other binary fail-closed回帰。 |
| `tools/dev-agent-harness/README.md` | documentation | excluded | plan-provision usage、JSONL semantics、read-only boundary、writer atomicity上限、executor非対象。 |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

`planned_implementation_files = 2`、`file_score = ceil(2 / 3) = 1`、`planned_implementation_lines = 340`（test/docs除外）、`line_score = ceil(340 / 200) = 2`なので、`estimate_points = 2`。手書きimplementation + testは約810行で、AC-9の700〜1,200行目安内である。

## 実装手順

1. `internal/provision`にheader/user/directory/serviceの固定record型、root/logical path validator、containment-aware mappingを置く。Configを再利用して3 user、4 directory、3 serviceのdesired stateを固定sliceへ構築する。
2. 全11 recordをmemoryでvalidateし、固定struct field orderでJSONL bytesへserializeする。validator後のsingle writer callだけを公開し、writer error時は再試行しない。
3. provision unit testを先に完成させ、byte-for-byte output、decoded record不変条件、全negative root/path/order/service/mapping mutation、target-root snapshot不変を固定する。
4. `internal/command`に`plan-provision --config PATH --target-root PATH`を接続し、safe error、stdout/stderr、exit、short/error writerをtestする。check-configと他5 binaryのfail-closed契約を回帰する。
5. READMEを更新した直後にroot `make lint-docs`を実行する。candidate固定前にharness `go test ./...`/`make check`/`make distcheck`、指定root check、`git diff --check`を実行し、REF-4 bindingを含むcandidate evidenceをMainへ渡す。

## 検証計画

- `tools/dev-agent-harness`で`go test ./...`を実行し、provision testでvalid Config/rootのJSONL byte equality、11行、headerと10 actionのdecode済みfield/sequence、repeat determinismを確認する。root（empty/relative/non-clean/NUL）とlogical/audit/containmentの拒否も観測する。
- injected writerでsuccessful single write、short write/error、encode前validation errorを観測する。writer failureはnon-zero・no retryとし、physical stream prefixを回復不能な限界としてcase evidenceへ記録する。
- temporary target-root treeを計画前後にsnapshot比較し、manifest生成が作成/変更しないことを確認する。package source/diffを確認し、process/network/IPC/systemd/filesystem mutation APIを導入していないことをevidence-reviewする。
- command testでvalid/invalid config、不足/余剰/順不同引数、invalid root、stderr非漏洩、stdout空、既存check-config/5 binaryの拒否を検証する。order、disabled/stopped、containment、writer-error guardを弱めるnegative mutationで各対応testが失敗することを候補証跡に残す。
- `go test ./...`、harness `make check`、`make distcheck`、README後のroot `make lint-docs`、candidate前の`PYTEST_ADDOPTS=--ignore=worktrees make check`、`make task-check TASK=TASK-0036`、`git diff --check`を実行する。network/VPS/sudo/PAM/systemctl/live-e2eは対象外として未実施理由を記録する。

## 未解決事項

- なし。JSON field/Go型名はここで固定した観測fieldと順序を満たす範囲でDEVが選ぶ。任意writerに対する完全なstdout transactionは不可能であるため、single write前の全record validation/serializationとnon-retryを本Taskの原子化契約とする。この限界を超える要求（OS-level atomic stdoutまたはexecutor）はMainが別Taskとして再審査する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] 見積もりが規則どおりである。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0036`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
