---
task_id: "TASK-0035"
change_class: "product"
status: draft
planner_agent: ""
approved_by: ""
approved_at: ""
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
planned_implementation_files: 3
planned_implementation_lines: 480
estimate_points: 3
---

# TASK-0035 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | setup専用のサブコマンド分岐を追加する。成功時は固定語彙だけの一行summary（version 1、default deny、validated）をstdoutへ出し、解析済み設定は返却しても表示しない。 | `cmd/dev-agent-harness-setup/main.go`、`internal/command/`、`internal/config/` | 4 | validation errorは分類だけをstderrへ出してnon-zeroとし、open/read/parse/validationのいずれでも設定値・path・user名・入力本文を出さない。 |
| AC-2 | bytesを上限内で取得後、token走査で各objectのkey集合を確認して重複を検出し、別decoderでstrictな型decodeを行う。root値の完了後はEOFだけを受理する。未知field、version不一致、syntax/duplicate/trailingは区別した内部error classにする。 | `internal/config/` とそのtestdata・テスト、`internal/command/` | 1, 2, 4 | token不整合、duplicate、unknown、second value、非空白末尾はすべて拒否する。CLIは安全な分類名だけを表示し、decoder由来の入力断片を透過しない。 |
| AC-3 | V1型はpaths/users/networkを明示し、decode後に必須field、絶対かつ`filepath.Clean`と同値のpath、3 userの相異、Linux useradd互換の安全な部分集合、`network.default == deny`を検証する。allowlist型fieldはV1に導入せず、欠落または空値を許可へ写像しない。 | `internal/config/`、既存`config/harness.json.example.in` | 2, 3 | 未設定・不正値はsemantic classで拒否し、将来versionまたはprovider固有の黙示的fallbackを設けない。 |
| AC-4 | Linuxのfile descriptorを一度だけ`O_RDONLY|O_CLOEXEC|O_NOFOLLOW`でopenし、同じFDに対するread前後の`fstat`でregular file、group/world non-writable、64 KiB以下、同一dev/inodeと検査時属性を確認する。bounded readerで上限超過を検出し、pathを再open/statしない。 | `internal/config/` とテスト | 1, 3 | open、fstat、read、再fstatの失敗または比較不能はfile-policy/I/O classで拒否する。symlink、directory、FIFO等を読まず、外部通信・IPC・OS操作は開始しない。 |
| AC-5 | valid fixtureと分類別invalid fixtureをtable-driven unit testへ割当て、CLIのexit/stdout/stderrも別testで検査する。各negative caseは「受理されたら失敗」の独立assertionを持ち、guardを受理側へ反転したmutationで当該testがredになることを候補証跡として残す。 | `internal/config/*_test.go`、`internal/command/*_test.go`、`internal/config/testdata/` | 3, 4 | fixture不足、分類の混同、mutationでgreenのまま、またはテスト弱体化はAC未達としてFAILにし、後続gateへ進めない。 |
| AC-6 | 既存の共通`Run`を通常binaryのfail-closed契約として残し、setupだけが`check-config`を許す専用entrypointを使う。`--help`/`--version`の既存出力を保持し、未知subcommand・引数不足/余剰は拒否する。 | `cmd/dev-agent-harness-setup/main.go`、`internal/command/` | 4, 5 | setupの通常起動、他5 binary、credential helperを操作可能にしない。回帰testが成功起動またはhelp/version差分を検出したらFAILとする。 |
| AC-7 | configure生成済み例を入力として使い、unit、既存make target、dist tarball内のconfigure/build/check、`DESTDIR` install後の配置済みexample検証を順に実行する。`configure`/`Makefile.in`は変更も再生成もしない。 | tests、`config/harness.json.example.in`、`README.md` | 5, 6 | commandまたはinstall treeの検証が失敗した場合は配布契約FAIL。live環境を要求せず、生成物・cache・stage rootはGit管理外へ隔離して削除可能にする。 |
| AC-8 | 標準libraryのみ、手書きproduction 480行、手書きtest約430行（fixture/docs/生成物を除外）の約910行を上限内の目標とする。security boundaryまたは1,200行超過見込みは実装を止めMainへ戻す。 | 全許可pathのみ | 全工程 | 許可path逸脱、外部module、外部通信、Credential、IPC、OS変更、configure再生成はscope breachとしてFAIL。Mainの別Task判断なしに分割・追加しない。 |

## 関連Wikiと判断

- このTaskはKakesu本体外の harness に限定する。Mainが完了後に、V1 strict decode、default deny、危険な設定file拒否を意味Wikiへ取り込む。
- REF-3はMainによりcommit `14af73e`の実値`go 1.24`へ再固定済みであり、外部Go moduleは追加しない。

## 補足設計

### 代替案と不採用理由

- `encoding/json`だけのdecode: duplicate keyを最後の値へ上書きしてしまうため不採用。先行token走査とstrict typed decodeを組み合わせる。
- path文字列への`Lstat`後の通常open: 検査対象と読取対象が分離されるため不採用。同一FDの`O_NOFOLLOW` openと前後`fstat`に限定する。
- JSON Schema generator/YAML/external parsing library: REF-3の標準library境界、依存と生成物を増やさない判断に反するため不採用。
- allowlist/provider/credential fieldの先取り: 追加のsecurity boundaryとなるため不採用。V1はnetwork default denyのみを契約にする。

### 責務・境界・不変条件

- `internal/config`だけが設定bytes、strict JSON、V1 semantic、FD file policyを所有する。返すerrorは内部classと安全な固定messageだけで、入力由来文字列を含めない。
- `internal/command`は引数・writer・exit codeの変換だけを所有する。stdoutは成功summary専用、stderrは安全な失敗分類専用とし、両者へ設定をserializeしない。
- setup binary以外は既存`Run`のfail-closed面を維持する。設定検証はread-onlyで、network、credential、IPC、persistent state、OS状態を触らない。
- ファイル検査はpathではなくopen済みFDを根拠にする。全ての異常・比較不能・未対応platform条件は許可でなく拒否へ倒す。

### 移行・互換性

- JSON versionは整数`1`だけを受理し、未知versionに互換fallbackを作らない。将来のfield/versionは明示migrationを要する。
- `dev-agent-harness-setup --help`と`--version`、他binaryの通常起動拒否は既存文言・exit契約を回帰testで固定する。新規`check-config --config PATH`だけを追加する。
- 既存configure生成exampleのfield構成はV1型と一致するため、templateを変更せず入力として検証する。configure script・Autoconf入力・配布targetは変更しない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `tools/dev-agent-harness/internal/config/config.go` | implementation | 380 | V1型、FD基準bounded reader、duplicate/strict/trailing decode、semantic validation、安全なerror class。 |
| `tools/dev-agent-harness/internal/command/command.go` | implementation | 95 | setup専用のargument/exit/output adapterと既存共通fail-closed面の維持。 |
| `tools/dev-agent-harness/cmd/dev-agent-harness-setup/main.go` | implementation | 5 | setup専用entrypointへ接続。 |
| `tools/dev-agent-harness/internal/config/config_test.go` | test | 320 | valid/invalid、FD policy、strict JSON、semantic、mutation検出。 |
| `tools/dev-agent-harness/internal/command/command_test.go` | test | 110 | CLI writer、exit、非漏洩、help/version、fail-closed回帰。 |
| `tools/dev-agent-harness/internal/config/testdata/` | fixture | excluded | valid/invalid JSONとfile-policy test用素材。 |
| `tools/dev-agent-harness/README.md` | documentation | excluded | `check-config`、非作用、install後example検証を記載。 |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

`planned_implementation_files = 3`、`file_score = ceil(3 / 3) = 1`、`planned_implementation_lines = 480`（test/fixture/docs除外）、`line_score = ceil(480 / 200) = 3`なので、`estimate_points = 3`。手書き実装とtestの合計目安は910行であり、AC-8の700〜1,200行目安内である。

## 実装手順

1. `internal/config`にV1の値型と入力値を露出しないerror classを置き、FD open/read前後検査、64 KiB上限、JSON token duplicate scan、strict typed decode、trailing判定を実装する。
2. schema/semantic validationを追加し、path、user、network denyと必須fieldを検査する。allowlistやprovider固有の拡張は入れない。
3. config unit/fixture testを先に完成させ、各拒否分類と受理反転mutationに対する検出力を記録する。
4. setup commandだけへ`check-config --config`を配線し、固定成功summary・安全なstderr・exit codeをwriter注入でtestする。既存共通commandと他binaryは変更後もfail-closedと確認する。
5. configure済みexample、`DESTDIR` install tree、READMEの利用方法を一致させる。`configure`、`configure.ac`、`Makefile.in`は編集・再生成しない。
6. `go test ./...`、`make check`、`make distcheck`、install後exampleのsetup検証、rootのpreflight/task-checkを候補証跡へ記録する。

## 検証計画

- `tools/dev-agent-harness`で `go test ./...` を実行し、table-driven casesでvalid、unknown、duplicate、version、trailing、path、user、network、file type、permission、sizeを観測する。各negativeはclass、non-nil error、入力非露出をassertする。
- command testで成功stdoutを固定し、stdout/stderr双方にfixture本文、configured path、user名、credential風文字列がないことを確認する。invalid caseはnon-zero、stdout空、分類済みstderrとする。
- 既存全binaryへ`--version`、`--help`、通常起動を実行し、setupの新subcommand以外が従来どおり拒否されることを確認する。
- `./configure`後に `make check` と `make distcheck` を実行する。別の一時`DESTDIR`へinstallし、そこに配置された`harness.json.example`をbuild済みsetupの`check-config`へ指定する。生成物、cache、stage rootはGit管理外に残す。
- rootで `make task-preflight TASK=TASK-0035` と `make task-check TASK=TASK-0035` を実行する。`git diff --check`も予定する。ネットワーク、VPS、sudo/PAM、service起動は実行しない。
- candidate evidenceには各caseのcommand、fixture/environment、cache条件、exit、artifact digest、未実施理由を記録する。独立REVIEW/QAは同一candidate commit/treeから開始する。

## 未解決事項

- なし。TASKが許容する範囲で、内部error class名と成功summaryの正確な固定文字列はDEVが上記非漏洩不変条件を満たすよう選択し、candidate証跡とtestへ固定する。

## main Agentレビュー

- [ ] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [ ] 設計観点と代替案を検討している。
- [ ] QA_PLANがTASK-firstで独立作成されている。
- [ ] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0035`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
