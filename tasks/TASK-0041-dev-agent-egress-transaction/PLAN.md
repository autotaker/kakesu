---
task_id: "TASK-0041"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "in-memory transaction、注入resolver、既存policy/capability APIの接続に限定し、実Credential、file、network、TLSを扱わないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T06:12:29Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T06:12:11Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0041 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `egresspolicy`に既存allow判断と同じ評価結果からcanonical provider、repository、operation、destination hostを返すscope APIを追加し、既存`Authorize`を変えない。 | `tools/dev-agent-harness/internal/egresspolicy/` | 1 | denyはzero scopeと既存固定errorに集約し、transactionへ曖昧なscopeを渡さない。 |
| AC-2 | transactionのRules、Subject、Request、PreparedRequest、resolver/Forwarder境界を定義する。生成時はnil依存とCredential最大長を検証し、non-nil zero policy/RegistryはExecuteで判定する。入力Authorization/body sliceは所有権を持たない。 | `tools/dev-agent-harness/internal/egresstransaction/` | 2 | nil依存又は範囲外設定は固定Rules error、non-nil zero policy/RegistryはExecute時の固定denyで拒否する。 |
| AC-3 | `Execute`はpolicy allow、provider別Authorization抽出、policy scopeだけから構成したcapability requestのConsumeを順に実行する。 | `tools/dev-agent-harness/internal/egresstransaction/` | 3 | policy/auth失敗ではConsume/resolver/Forwarderを呼ばず、Consume失敗ではresolver/Forwarderを呼ばない。caller入力でscopeを補完しない。 |
| AC-4 | Consume成功後にだけcanonical provider/repositoryでresolverを一回呼び、Credentialを検証してBearer headerへ置換し、Forwarderを一回呼ぶ。 | `tools/dev-agent-harness/internal/egresstransaction/` | 4 | resolver、Credential検証、Forwarderの失敗は固定execute errorとし、retry、capability復元、並行時の複数到達を許さない。 |
| AC-5 | Credential-bearing PreparedRequestはtransaction内部から注入済みForwarderへ同期的にだけ渡し、Executeは値を返さない。request copyとcanonical scopeを渡し、Agent入力Authorizationとhandleを残さない。 | `tools/dev-agent-harness/internal/egresstransaction/`、`tools/dev-agent-harness/README.md` | 5 | error、Transaction state、Execute戻り値から機微入力及びresolver/Forwarder detailを漏らさず、transactionにI/O、環境、process、network、DNS、TLSを導入しない。 |
| AC-6 | policy scope、互換性、両provider、Authorization境界、拒否順序、Credential、Forwarder handoff、入力不変、non-leak、並行1-useをunit testで網羅し、candidate diffの実測を監査する。 | `tools/dev-agent-harness/internal/egresspolicy/`、`tools/dev-agent-harness/internal/egresstransaction/`、`tools/dev-agent-harness/README.md` | 6 | race、失敗順序違反、許可外の外部作用/変更path、base...candidateの追加＋削除が1,200超なら候補を完了扱いにしない。 |

## 補足設計

### transaction境界と順序

- `egresspolicy`だけがURLのallow評価とcanonical scope導出を所有する。transactionはscopeを再解析・正規化せず、その全fieldをcapability消費に用いる。non-nil zero policy/RegistryはExecute時に固定denyとする。
- RequestはAuthorization値をsliceで受け、provider別に許す値がちょうど一つのcapability handleである場合だけ受理する。余分な値、空白、改行、別schemeは入力段階で拒否する。
- `Execute`の成功経路は allow → Authorization抽出 → Consume → resolver → Credential検証 → PreparedRequest構築 → Forwarder同期呼出である。Consume後の失敗は消費済みのままにする。
- resolverはcanonical provider/repository以外を受け取らず、成功結果は上流Bearerへ変換する。PreparedRequestはbodyを独立copyし、capability handle又は元Authorizationを保持しない。Credential-bearing値はForwarderの同期呼出中だけ渡し、Executeは値を返さずTransactionも保持しない。

### 代替案と不採用理由

- transaction側でURLからprovider scopeを再構成する案は、policyと異なる解釈を作るため採用しない。
- resolverをConsumeより前に呼ぶ案は、未認可requestを秘密処理へ到達させるため採用しない。
- resolver/Forwarder失敗時のcapability rollback又はretryは、一回消費と副作用境界を曖昧にするため採用しない。
- HTTP transport、秘密保管、Credential取得実装を同梱する案は、対象外のfile/network/TLS境界を広げるため採用しない。

### 責務・境界・不変条件

- `egresspolicy`はcanonical allow scope、`capability`は既存の完全一致一回消費、`egresstransaction`は両者と注入resolver/Forwarderの接続だけを担当する。Forwarderはbroker内のtrusted upstream adapterであり、Agent入力を受ける側へPreparedRequestを返さない。
- 固定errorはURL、body、Subject、repository、handle、Credential、resolver detailを含まない。
- caller所有のRequest/bodyと入力Authorizationは変更せず、Forwarderへ渡す値にはcanonical scopeと上流送信用の値だけをコピーして持たせる。
- 同一1-use handleの競合は既存RegistryのConsumeを唯一の消費点とし、resolverとForwarderへ到達するのは一件だけとする。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/egresspolicy/` | existing allow評価からcanonical provider scopeを返すAPIと互換性検証。 |
| `tools/dev-agent-harness/internal/egresstransaction/` | Rules、request/PreparedRequest値型、厳密なAuthorization抽出、順序付きExecute、resolver/Forwarder/Credential境界とunit tests。 |
| `tools/dev-agent-harness/README.md` | transactionの責務、capabilityから上流Bearerへの置換、実Credential/proxy/network非対象を記載。 |

## 実装手順

1. policyの既存allow評価を壊さず、transactionが消費できるcanonical scope APIを追加する。
2. in-memory依存だけを受けるtransactionの公開値型、nil依存とBearer長のRules validation、固定安全errorを定義する。
3. provider別Authorization抽出と、policy scopeに束縛したConsumeからresolver/Forwarder呼出までのfail-closed順序を実装する。
4. Credential検証、PreparedRequestのコピー、同期Forwarder handoffと非漏洩を実装し、READMEを境界に合わせて更新する。
5. unit testsとrace検査で成功、拒否順序、並行一回消費、不変性、非漏洩を確認する。

## 検証計画

- `egresspolicy`で二つのallow surfaceのcanonical scopeとdeny時zero scope、既存`Authorize`のdecision/error互換性を確認する。
- transactionでRules/nil依存拒否、non-nil zero policy/RegistryのExecute deny、両providerの厳密なAuthorization、policy・capability・resolver・Forwarderの各失敗における呼出順序と固定errorを確認する。
- resolverの入力、Credentialの長さ・ASCII制約、成功時のBearer置換、Forwarderへの同期一回handoff、bodyを含む入力不変、handle/入力Authorization/詳細の非保持・非漏洩を確認する。
- 同一1-use handleの並行Executeでresolver/Forwarder呼出が各一件だけであり、race detectorがdata raceを報告しないことを確認する。
- DEVは指定されたpackageのrace test、harnessの`make check`/`make distcheck`、差分の空白検査、base...candidateの`git diff --numstat`追加＋削除合計を実施する。candidate固定後のroot `make check`とpost-merge `make task-check TASK=TASK-0041`はMainの完了経路で実施する。実Credential、file、network、TLSは対象外であり検証PASSの根拠にしない。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| policyとtransactionのscope解釈が分岐する | policyの単一評価結果をscope APIとして渡し、scope再解析を行わない | transaction側でURL又はrepositoryの補完・再解析が必要になれば停止してMainへ戻す。 |
| 未認可requestがresolver/Forwarderへ到達する | 段階的な呼出順序と拒否経路のunit tests | Consume成功より前のresolver、又はvalid Credential完成前のForwarder呼出が必要になれば停止する。 |
| Credential又はhandleが不要な出力へ残る | 同期Forwarder handoff、Executeの値なしreturn、固定error、non-leak tests | 上流Bearer以外のfield/error/stateへCredentialを置く、又はhandle/入力AuthorizationをForwarder入力へ残す必要が生じたら停止する。 |
| capability消費後の失敗で再利用が可能になる | rollbackなし、retryなし、並行1-use検証 | capabilityの復元又は別の再試行意味論が必要になれば別Taskへ戻す。 |

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] policy scope、capability consume、resolver/Forwarderの責務とfail-closed順序が一意である。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

## planning review

PASS — TASK-firstのACは一意に実装可能で、policy/scopeの単一評価、Authorization sliceの厳密な一値境界、Consume → resolver → valid Credential → 同期Forwarderの順序、消費後失敗時のrollback/retryなし、credential-bearing PreparedRequestのtrusted Forwarder限定handoffを整合して定義する。QA_PLANはcandidate-boundで、race testのfocused-rerunをQA-004の一回だけに限定し、他を証跡監査とする。許可pathおよびbase...candidateの追加＋削除1,200行上限もQA-006で監査する。
