---
task_id: "TASK-0041"
status: complete
reviewer_agent: "reviewer-agent-terra-medium"
decision: pass
reviewed_at: "2026-08-01T06:27:39Z"
---

# TASK-0041 REVIEW RESULT

## 監査対象

- Task ブランチ の 案 diff と DEV の `make check` 証跡を独立に監査する。
- candidate_commit は HANDOVER の一箇所だけで管理する。

## 監査したDEV証跡

| コマンド/テスト | 結果 | 備考 |
|---|---|---|
| DEVのfocused race test、harness `make check`、最終tree `make distcheck`、root `make check`、文書lint、`git diff --check` | `PASS` | HANDOVERのcandidate-bound表を候補diffと照合した。これらのテスト／外部コマンドはレビューでは再実行していない。読み取りによる候補diffの空白検査はPASS。変更はREADME、`internal/egresspolicy/`実装・scope test、`internal/egresstransaction/`実装・unit testの5ファイルで、673行追加・19行削除、計692行。 |

## 受け入れ条件の確認

| 条件 | 結果 | 根拠 |
|---|---|---|
| AC-1 policy scope / Authorize互換 | `PASS` | `Policy.Evaluate`は従来のallow評価を一度だけ行い、GitHubには`github`／canonical repository／`github-rest-read`／`api.github.com`、OpenAIにはrepositoryなし／`openai-responses-text`／`api.openai.com`を返す。denyはzero Scope、`DecisionDeny`、既存の`request-denied`へ集約され、`Authorize`はEvaluateのdecision/errorへ委譲する。 |
| AC-2 Rules、Subject、入力所有権 | `PASS` | `New`は全依存nilと1〜4,096外のCredential上限を固定`invalid-rules`で拒否する。nil／non-nil zero Transaction依存はExecuteで固定denyとなる。SubjectはConsume requestへagent instance・non-root UID・workspaceをそのまま渡し、registryの完全一致検証に束縛する。bodyはForwarder用にだけcopyし、Authorization sliceは読み取りのみで保持・変更しない。 |
| AC-3 allow→厳密Authorization→Consume | `PASS` | Executeはpolicy Evaluate、provider別一値scheme抽出、Evaluate由来の全scope fieldを使う`capability.Request`のConsumeの順で進む。OpenAIは`Bearer cap_...`だけ、GitHubは`Bearer cap_...`又は`token cap_...`だけを受け、大小文字、余分な空白、複数値、改行、別schemeをConsume前に拒否する。policy/auth denyはcapability/resolver/Forwarderを呼ばず、Consume denyはresolver/Forwarderを呼ばない。 |
| AC-4 Consume後のCredential解決と一回性 | `PASS` | 成功Consume後だけcanonical provider/repositoryでresolverを一度呼び、空、設定上限超過、space/tab/newline/control/non-ASCII Credentialを拒否する。resolver/Forwarder失敗にretry又はrollbackはなく、全実行失敗は固定`transaction-denied`となる。registryのmutex下の原子的Consumeと16並行worker testは、同一1-use handleでresolver/Forwarderへ到達するのが各一件だけであることを検出する。 |
| AC-5 trusted Forwarder handoff / non-leak | `PASS` | PreparedRequestはmethod、raw URL、content type、独立body copy、Evaluate由来Scope、実Credentialの`Bearer `値だけを持つ。capability handleと入力Authorizationは残らず、Executeの戻り値はerrorだけで、Transaction fieldsにもrequest／handle／Credential状態はない。sourceはmemory-boundaryのみでfile/environment/process/network/DNS/TLSを導入せず、失敗detailも返さない。 |
| AC-6 tests、範囲、DEV証跡 | `PASS` | candidate testsはscope導出、Authorize互換、両provider成功、scheme境界、policy/scope/capability/resolver/Forwarder deny、Credential検証、消費順序、input不変、non-leak、並行1-useを覆う。HANDOVERのDEV証跡は指定race test、harness check/distcheck、root check、lint、diff checkのPASSを記録する。許可外path・test削除・期待値緩和はなく、差分は5パス・692行で上限1,200以下。 |

## 指摘

- なし

## 結論

`pass` — blocking findingなし。HANDOVERに一箇所だけ記録されたcandidateに固定して、製品diff、source/test、DEV candidate-bound check証跡を独立監査した。実Credential source、Forwarder実装、file/network/TLS/DNS、HTTP通信はTask対象外であり、本レビューはそれらの実行保証を主張しない。
