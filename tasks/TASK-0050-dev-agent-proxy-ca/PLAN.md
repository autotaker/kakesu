---
task_id: "TASK-0050"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "標準libraryだけの新規in-memory CA Authorityとhermetic TLS fixtureに閉じ、既存製品packageを変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T11:58:06Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0050 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules` は単一certificate PEM、対応する単一ECDSA P-256 private-key PEM、非nil Clockだけを受ける。PEM block/trailing、自己署名、CA constraints/CertSign、key一致、現在有効性と15分以上の残存を`New`で検証する。 | `tools/dev-agent-harness/internal/proxyca/` | 1 | encrypted/multiple PEM、RSA/Ed25519、弱い・不一致key、typed nil、nil/zero receiverはfixed non-leak errorへ畳み、parser/PEM detailを公開しない。 |
| AC-2 | Authorityはparse済みCA certificate/signer、certificate-only PEMのcopy、Clockだけをimmutableに保持する。`PublicCertificatePEM`は毎回新しいcertificate-only copyを返し、private PEM/signer/leaf keyは公開API・formatへ出さない。 | `tools/dev-agent-harness/internal/proxyca/` | 1–2 | input mutation/alias、public outputへの追加block又はprivate material、format/errorへのsubject/serial/host detailは固定error又は空安全値にする。 |
| AC-3 | `Issue`はbyte-for-byte exactな`api.github.com`又は`api.openai.com`だけを許し、case差、port、末尾dot、wildcard、IP、空、control/non-ASCIIを署名前に拒否する。host補正、cache、retryを置かない。 | `tools/dev-agent-harness/internal/proxyca/` | 3 | 非許可hostはserial/key/certificateを生成・返却せずfixed errorだけを返す。 |
| AC-4 | 許可host呼出しごとに新しいECDSA P-256 leaf keyとnonzero random 128-bit serialを生成し、single exact DNS SAN、empty CN、ServerAuth、DigitalSignature、non-CA constraintsだけを持つleafをCAで署名する。NotBeforeはClockから最大5分backdate、NotAfterは15分以内かつCA期限前に制限する。 | `tools/dev-agent-harness/internal/proxyca/` | 3–4 | Clock/RNG/key/署名/期限演算の任意失敗はpartial certificateを返さずfixed non-leak errorにする。IP/email/URI SAN、ClientAuth、CertSignは発行しない。 |
| AC-5 | `tls.Certificate`はleaf→CA chain、parse済みLeaf、call-local leaf private keyを持つ。certificate/key/serial/bufferをcall間で共有せず、公開CAを用いるTLS 1.2/HTTP1.1相当の`net.Pipe` handshakeとhostname verifyで両hostの実効性を確認する。 | `tools/dev-agent-harness/internal/proxyca/` | 4 | wrong host、expired CA、未許可host、handshake/verify失敗はfail closedとし、TLS listener、SNI routing又はcacheで補わない。 |
| AC-6 | in-memory CA/PEMとClock、`net.Pipe`を使うrace suiteでPEM/key/validity拒否、copy/non-leak、host exact、extension/validity/chain、handshake、並行unique serial/keyとSAN隔離を検出する。 | `tools/dev-agent-harness/internal/proxyca/`、`tools/dev-agent-harness/README.md` | 1–5 | fake/in-memory PASSを実CA file、trust install、実client/provider、listener/CONNECT、OS trust storeの成功根拠にしない。 |

## 責務・境界・不変条件

- Authorityはbroker memory内のCA入力検証、公開CA export、2 host限定leaf発行だけを所有する。TASK-0049 Handler、Exchange、Policy、credential/transportは変更せず、listener/CONNECT、file lifecycle、trust installを合成しない。
- 長寿命stateは検証済みCA certificate/signer、certificate-only public PEM、Clockだけである。入力PEM、CA private PEM、leaf private key、leaf certificate、serialとcall-local bufferは保持しない。
- 成功順序は Rules/Authority検証 → exact host判定 → random serialとP-256 key生成 → fixed extension/validity leaf構築 → CA署名/parse → leaf→CA chain返却である。失敗後retry、host正規化、certificate/key cacheを持たない。
- public exportはCA certificate blockだけの独立copy、Issue結果はTLS server用途のleaf keyを含むが、Authorityのerror/format/public exportには秘密、subject、serial、host、parser detailを出さない。並行callはmutable stateを共有しない。
- READMEはmemory-only Authorityと公開CA/leafの責務、listener/file/trust/live E2E未確認境界だけを既存説明へ追記する。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/proxyca/` | immutable Authority Rules/New、strict PEM/CA/key検証、certificate-only export、2 host限定ECDSA P-256 leaf発行、fixed error、in-memory/net.Pipe hermetic race testsを追加する。 |
| `tools/dev-agent-harness/README.md` | broker memory内のproxy CA Authorityの範囲、公開CAのみを返す境界、およびfile/listener/trust/live E2Eが対象外であることを追記する。 |

## 実装手順

1. 固定error、Clock、Rules/Authorityとsingle-PEM/typed-nil/receiver検証を定義し、certificate parse、自己署名、CA constraints、ECDSA P-256 key一致、Clock有効期間を試験する。
2. Authorityにcertificate-only public PEMの独立copyだけを保存・返却させ、input/output ownership、format/error non-leak、private material不在を検証する。
3. exact two-host gate、call-local random 128-bit serial/P-256 key、15分以内かつCA期限内のvalidity、最小leaf extensionを実装し、拒否が署名前であることを試験する。
4. leaf→CA chainとparse済みLeafを組み、`net.Pipe`のTLS 1.2/HTTP1.1相当handshake、hostname verify、wrong-host/expired CA拒否、並行unique serial/key/SAN隔離をrace fixtureで確認する。
5. READMEを責務境界に合わせ、`GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxyca`、harness `make check`/`make distcheck`、README変更時のTask worktree `make lint-docs`、`git diff --check`、許可2pathとbase...candidate追加＋削除1,000行以下をcandidate前に確認する。candidate launcherのroot `make check`は一回だけとし、planning/candidate/completion以外のcommit、新規gate/process/digest/candidate重複記録を作らない。

## 検証計画

- focused testの `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxyca` で、single PEM/trailing、CA/self-sign/constraint/key/validity、input/public copy、fixed non-leak、exact host拒否、leaf extension/15分validity/chain、TLS handshake/hostname verify、concurrent uniquenessとraceを検出する。
- fixtureはtest生成CA、in-memory PEM、注入Clock、`net.Pipe`だけを使う。実filesystem秘密、listener port、network、実trust storeへ到達しない。
- harness `make check`/`make distcheck`、Task worktree `make lint-docs`、candidate launcherのroot `make check`はDEVのcandidate-bound証跡として監査し、QAのfocused rerun又はhermetic PASSをlive TLS配置の代替にしない。

## 停止条件

- CA/leaf file lifecycle、secret storage、trust install、listener/CONNECT/SNI、HTTP handler/broker composition、config/process wiring、実network/clientを追加する必要があれば停止する。
- RSA/Ed25519、intermediate/multiple CA、wildcard/任意host、IP/client SAN、mTLS、cache、可変lifetime、CRL/OCSPが必要ならstrict two-host contractを広げず停止する。
- 既存Handler/Exchange/Policy/credential/transportの変更、許可外path、外部dependency、又は1,000行上限超過が必要ならMainへ戻す。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0050`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
