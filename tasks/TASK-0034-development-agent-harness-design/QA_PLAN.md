---
task_id: "TASK-0034"
change_class: safety_contract
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T08:45:10+10:00"
revision: 3
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0034 QA PLAN

## TASK-first baseline

この計画はDEV開始前に、`TASK.md`の`planning input packet`だけを入力として作成した。`PLAN.md`は読まず、
開始条件にも使わない。観測されたQA roleの指定は`qa`、`gpt-5.6-terra`、`medium`であり、
[`.codex/agents/qa.toml`](../../.codex/agents/qa.toml)の正規契約と一致する。

本Taskは製品成果物・宣言済み製品依存を変更しない`safety_contract`である。ここでいうPASSは、完成した
設計文書がTASKの静的な契約を満たすかの計画レビュー用証跡であり、製品DEV、製品REVIEW/QA PASS、
`QA_RESULT.md`の製品PASS、実VPSへの導入を意味しない。

## 共通の実施条件

- 静的文書契約は`evidence-review`とする。QAは同一candidateのcommit/tree、全差分pathとdigest、
  文書内のsection/flow/state-transition/reference、対象検査のcommand・環境・cache条件・exit・出力digest、
  negative検出能力とtest/検査の弱体化有無をTASK-first基準と突合する。
- `docs/glossary.yml`は通常の予定pathである。用語generatorはcandidate上で二回実行し、各exit、各回前後の
  path一覧/digest、第二回後のdriftなしを記録する。candidateの初回生成差分は`docs/glossary.yml`だけであり、
  `docs/99-glossary-index.md`は再生成後もplanning base比で内容不変でなければならない。従って
  `safety_contract_generated_paths`は空配列である。
- 安全契約のcandidateに製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、
  又は製品外部挙動が含まれるなら、静的PASSを停止して製品変更へ再分類する。candidate/tree/digestの不一致、
  許可外path、ACの意味不明・欠落・矛盾、又は高リスクの実際の挙動を文書だけで主張する場合もPASS不可である。
- 実UbuntuのOS identity・filesystem/process/socket/network隔離、Tailscale tailnet/Serve/Grants、Passkey、
  GitHub/OpenAI等の外部service、実credential、実restart/rollback/cleanupを要する事実は`live-e2e`でのみ
  確認する。本Taskではinstall・設定・秘密配置・サービス起動が対象外なので、該当ケースは将来実装Taskまで
  `blocked/not-run`とする。`evidence-review`、mock、又は文書の記述で代替PASSにしない。
- `focused-rerun`は割り当てない。実装・fixtureがまだなく、認証、秘密、外部作用、永続化、失効/再生、
  restart/cleanupを含む受け入れ真実をhermetic・deterministic・boundedに完全再現できる根拠がないためである。
- FAIL候補は証拠に基づき`implementation_defect`、`qa_plan_defect`、`requirement_gap`、
  `environment_issue`、`regression`へ分類し、DEV起因とは仮定しない。最終分類はMainが行う。

## 受け入れ条件との対応（設計文書）

| ケースID | AC-ID | 観測方法 | `qa_execution_mode` / 理由 | 必要証跡 | fail-closed |
|---|---|---|---|---|---|
| QA-001 | AC-1 | 文書の冒頭、名称、component境界、依存関係、対象外、将来採用時の再審査を横断し、外部`Development Agent Harness`であってKakesu本体ではないことを一意に追跡する。Kakesu Plane/message/Authority/state/tabletopへの導入や製品artifact変更がないことを差分とnegative検索で確認する。 | `evidence-review` / これは文書上のscope・命名・非採用境界であり、candidate-bound静的証跡で判断できる。 | candidate commit/tree、path一覧/digest、該当section anchors、Kakesu製品境界の検索結果、`git diff --check`。 | 外部基盤と製品の区別、対象外、又は将来の別製品Task/re-review条件が欠落・矛盾する、製品pathが差分に混在する、又はcandidate束縛が不明ならPASS不可。 |
| QA-002 | AC-2 | owner/login user、agent user、Codex auth例外、Credential Broker/Egress Proxy/Approval Serviceのtrust boundaryを読み、filesystem、environment、process、socket、networkの各経路でagentが実秘密を取得できない設計上の遮断と、例外がmodel生成commandへ継承されないことを確認する。 | `evidence-review` / 要求は設計文書が境界を定義すること。実隔離の有効性はLVE-001へ分離する。 | component/identity matrix、全5経路のdeny根拠、例外のscope、secret storage/redaction/audit記述、negative経路の記述、candidate digest。 | identity又は経路が一つでも未定義、実credential/owner secretをagentへ渡す経路、Codex例外の継承範囲不明、又は文書だけで実隔離済みと主張する場合はPASS不可。 |
| QA-003 | AC-3 | `gh`、Agent codeのOpenAI API、Git Smart HTTP fetch/pullについて、Opaque capabilityの提示、Broker/Proxyだけでの短命credential置換、宛先・repo・operation検証、credential helperの境界をflowごとに追跡する。pushが通常のread flowへ迂回しないことも確認する。 | `evidence-review` / flowと制約の完全性は静的に監査できる。実通信/credential exchangeはLVE-002へ分離する。 | protocol/sequence図又は同等flow、各client→broker/proxy→upstreamの許可/拒否表、capabilityとcredentialの非同一性、REF-3/4/5/6への対応、negative flow、candidate digest。 | 3 flowのいずれか、宛先/repo/operation制限、credential非露出、又はpushとの分離が欠落する、恒久token/SSH keyをagentへ許す、参照追跡不能ならPASS不可。 |
| QA-004 | AC-4 | push requestの必須束縛属性、exact one-use消費、approval中にprocessを保持しないこと、agentによる明示的再実行、expiry/deny/cancel/stale/replay/TOCTOU/競合を含む非同期state machineを、正常・境界・negative transitionとして確認する。 | `evidence-review` / state machineの設計契約を読むケース。実grant発行・並行競合・永続化はLVE-003へ分離する。 | state/transition表、request/grant field表、one-use消費点、再実行/取消/expiryの操作記述、監査event/redaction、negative/replay/TOCTOU handling、candidate digest。 | 必須属性、one-use、非保持、明示的再実行、又はいずれかの失効/再生/競合処理が欠落・fail-open・曖昧ならPASS不可。承認だけでagent pushが自動継続する設計もFAIL。 |
| QA-005 | AC-5 | Approval UIのlocalhost bind、Tailscale Serve経由のtailnet HTTPS、Funnel無効化、Tailscale identity/Grantsと毎回のPasskey user verificationのAND条件、通知の非権限性を、設定責務・request flow・拒否分岐で確認する。 | `evidence-review` / 要求は構成・多層認証の設計定義。実tailnet/Passkeyの効果はLVE-004へ分離する。 | bind/ingress matrix、Serve/Funnelの明示設定、identity・Grant・PasskeyのAND truth table、notificationの権限なし根拠、REF-1/2/7対応、candidate digest。 | localhost/tailnet限定、Funnel拒否、identity/Grant/Passkeyのいずれか、又は通知非権限が欠落・OR化・曖昧ならPASS不可。文書だけで実tailnet公開面が検証済みと扱ってもPASS不可。 |
| QA-006 | AC-6 | network/credentialのdeny-by-defaultとfail-closed、threat model/assumptions、secret storageとCodex例外、audit redaction、revocation/device loss、stale/expired/replay/TOCTOU、障害時の停止・復旧・rollbackを、境界と所有責務に結び付けて確認する。 | `evidence-review` / 必要なのは安全契約の定義と網羅性。実障害注入/失効/復旧はLVE-005へ分離する。 | threat model、assumption/exclusion、deny/failure matrix、secret/audit/redaction policy、revocation/recovery runbook、各failureの停止状態と再開条件、candidate digest。 | fail-open、秘密の平文監査、脅威/前提不明、revocation・lost device・stale/expiry/replay/TOCTOU・復旧の欠落、又は復旧が権限を拡大する場合はPASS不可。 |
| QA-007 | AC-7 | 最小構成、段階導入、未決の実装選択、検証matrix、live VPS依存項目、後続Taskへの分割単位を確認し、未決を実装済みと扱わず、Kakesu採用判断がpendingで本Taskを拡張しないことを確認する。併せてterminology generatorを二回実行し、初回のcandidate差分が通常予定path `docs/glossary.yml`に限られること、二回目にdriftがないこと、`docs/99-glossary-index.md`がplanning base比で内容不変であることを確認する。 | `evidence-review` / roadmap/検証matrixと、決定的な文書生成のcandidate-bound証跡を静的に監査できる。matrixで列挙した実環境行はLVE-001〜005としてnot-runのまま残す。 | component/minimum scope、phase表、decision log、verification matrix、後続Task cut lines、live-only列挙とblocked理由、dependency-ready reconciliation、candidate digest、generator二回分のcommand/environment/cache/exit、各回前後のpath一覧/digest、base/candidate/再生成後の`docs/glossary.yml`とindex digest、空の`generated_paths`宣言。 | 最小構成・段階・未決・live-only・後続分割のいずれかが欠落、実環境項目を静的PASSで閉鎖、又はpending採用を本Taskの完了条件へ混入するならPASS不可。generator失敗、二回目のdrift、`docs/glossary.yml`以外の初回差分、indexのbase比内容差分、未宣言pathの生成/変更、又は非空`generated_paths`はPASS不可。 |
| QA-008 | AC-8 | REF-1〜REF-7の公式参照を文書上で特定し、Tailscale Serve/Grants、Git credential helper、GitHub App installation authentication、Codex sandbox/network/credential注意事項、Passkeyと各設計判断の対応を追跡する。 | `evidence-review` / 要求は公式参照と設計判断の静的traceabilityであり、このTaskの固定参照をcandidateと照合できる。 | reference table/links、REF-ID→判断/section mapping、取得・固定情報、参照不能時の扱い、candidate digest。 | 必須カテゴリ又は判断対応がない、非公式資料だけで代用する、参照と結論が矛盾する、又は取得不能を確認済み事実として扱う場合はPASS不可。 |

## 将来の実環境検証（本Taskでは `blocked/not-run`）

| ケースID | 対応AC | `qa_execution_mode` | 将来の観測 | このTaskでの状態とfail-closed |
|---|---|---|---|---|
| LVE-001 | AC-2, AC-6 | `live-e2e` | 承認済み隔離Ubuntuで別owner/agent OS identity、unit、namespace、filesystem、environment、process、socket、networkを実測し、agentから各実secret/Codex credentialへの到達を試みて拒否を確認する。 | `blocked/not-run`。VPS、実identity、secret fixture、許可、安全cleanupが対象外で未用意。静的境界記述、mock、又は未実施で実隔離PASSを出してはならない。 |
| LVE-002 | AC-3, AC-6 | `live-e2e` | 実GitHub App、Git Smart HTTP、`gh`、OpenAI endpointでcapabilityを使い、allowed destination/repo/operationだけが短命credentialへ置換され、denied host/repo/opとcredential抽出が拒否されることを確認する。 | `blocked/not-run`。外部service、実token、network policy、audit redaction、cleanupが対象外。外部credentialを共有環境へ置く、又は実upstreamなしにtransport PASSとする場合はfail-closed。 |
| LVE-003 | AC-4, AC-6 | `live-e2e` | 承認request永続化、expiry、cancel、deny、同一grantの同時再使用、old-SHA変化、force/delete、crash/restart/rollbackを実serviceで注入し、one-useと明示再実行を観測する。 | `blocked/not-run`。Broker/Approval実装、永続store、承認端末、破壊的操作の隔離/rollbackがない。正常push一回だけ、又はmockだけでone-use/TOCTOU PASSにしてはならない。 |
| LVE-004 | AC-5, AC-6 | `live-e2e` | tailnet内外からServe UIへ接続し、localhost backendのみ、Funnel無効、Tailscale identity/Grant拒否、Passkey UV必須、通知だけでは承認不能を実測する。 | `blocked/not-run`。Tailscale tailnet、Serve/Grants、スマートフォン、Passkey registration、isolated cleanupが対象外。ネットワーク公開、identity、又はUVを文書/スクリーンショットだけでPASSにしてはならない。 |
| LVE-005 | AC-6, AC-7 | `live-e2e` | device loss、credential revocation、broker/proxy/approval outage、audit redaction、restart、recovery/rollbackを承認済み実配置で実行し、停止・復旧後の最小権限を確認する。 | `blocked/not-run`。実配置、秘密、障害注入、restart/rollback手順が対象外。安全なcleanupと復旧判定が確定しない限り実行しない。 |

## 完成時の静的検査と証跡

安全契約の候補について、少なくとも次をcandidate-boundに記録する。実行コマンドは完成差分の実pathへ限定し、
対象外の製品検査を製品QA PASSの根拠にしない。

```sh
git diff --check <planning-base>...<candidate>
git diff --name-only <planning-base>...<candidate>
rg -n 'Development Agent Harness|Kakesu|Credential Broker|Egress Proxy|Approval Service|Opaque capability|one-use|Passkey|Funnel|Tailscale Serve|Grant|revocation|TOCTOU|live-e2e' docs/development/development-agent-harness.md
make task-preflight TASK=TASK-0034
git diff --name-only <planning-base>...<candidate> -- docs/glossary.yml
git diff --exit-code <planning-base>...<candidate> -- docs/99-glossary-index.md
uv run --project memory python scripts/validate-terminology.py --write
git diff --name-only
uv run --project memory python scripts/validate-terminology.py --write
git diff --exit-code
make check
make task-check TASK=TASK-0034
```

候補差分全pathは、承認済みPLANの`safety_contract_planned_paths`と
`safety_contract_generated_paths`の和集合へ束縛する。このTaskでは設計文書と`docs/glossary.yml`を前者、
後者を空配列として記録する。`docs/99-glossary-index.md`はgenerator再実行後もplanning base比で内容不変で
なければならず、候補差分にもgenerator差分にも含めない。generatorが`docs/glossary.yml`以外を変更した場合は
未宣言pathとしてfail-closedする。二回目実行後に差分が残る場合は未収束driftとしてPASS不可である。PLANの合格、
Reviewerの合格、又はこのQA計画そのものはQA開始の前提にしない。安全契約の完了は独立計画レビュー、
Mainの分類承認、対象検査、no-ff merge、およびcandidate/merge tree一致に従い、製品用の
REVIEW/QA PASSやWiki receiptで代替しない。

## 再計画・未実施時

- 実装後にTASK-first期待値とcandidateを照合する。期待値又は範囲を変える必要があれば、`qa_plan_defect`
  又は`requirement_gap`候補としてMainへ戻し、理由と承認を記録するまでPASSにしない。
- live-e2eケースは実装Taskで隔離環境、必要な外部権限、影響範囲、rollback/cleanup、具体的なresult証跡を
  再計画する。環境未準備は実装不具合とは決めつけず`environment_issue`候補とする。
- candidate/tree不一致、認証認可・秘密・IPC/設定/依存・lifecycle/error/fail-closedを含む変更、QA FAIL、
  または影響不明があれば`qa_carry_forward`を禁止する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | Main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | QA Agent | TASK-first初版。AC-1〜AC-8の静的`evidence-review`と、実Ubuntu/Tailscale/Passkey/external serviceを必要とする将来`live-e2e`を明示的に分離した。 | `main-agent approved` |
| 2 | 2026-08-01 | QA Agent | Dependency-ready reconciliation。ACの意味を変えず、`docs/glossary.yml`を通常予定path、`docs/99-glossary-index.md`を生成pathとして、generatorの二回実行・index生成・未宣言path・未収束driftをQA-007の静的証跡へ追加。 | `main-agent approved` |
| 3 | 2026-08-01 | QA Agent | 2回目のTASK-first reconciliation。ACの意味を変えず、初回generator差分を`docs/glossary.yml`だけへ限定し、indexのbase比内容不変、空の`generated_paths`、二回収束、未宣言path拒否へ改訂。 | `main-agent approved` |
