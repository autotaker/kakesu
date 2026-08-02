---
task_id: "TASK-0066"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T03:12:56Z"
revision: 4
implementation_reviewed_at: "2026-08-02T03:37:49Z"
expectation_changed: true
expectation_change_approved_by: "main-agent-sol-high"
---

# TASK-0066 QA PLAN

## 方針

この計画は `TASK.md` の planning input packet だけからDEV開始前に独立作成した。QAはHANDOVERが固定する同一 `candidate_commit` を対象にし、server と client の相互成功だけを信頼しない。公開CAでも失敗診断への露出、Authority境界、strict framing、接続寿命は安全性の検査対象である。

唯一の再実行は、candidate treeで一回だけ行う次のhermeticかつ上限付きのrace検査である。net.Pipe、fake Authority、fake dialerだけを使い、実socket、外部ネットワーク、ファイル、環境設定を使わない。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/connectsession ./internal/controlclient ./internal/egressservice
```

`make -C tools/dev-agent-harness check`、`make -C tools/dev-agent-harness distcheck`、candidate gateのroot `make check`、`git diff --check`はHANDOVERのcommand/result記録と、成功時だけcandidate commitを作るgate実装をcandidate-bound証跡として独立監査する。QAはraw logやartifact digestを追加要求せず、これらを再実行してPASSを置き換えない。command/result欠落、candidate不一致、又はgateを通らないcommitなら証跡不足としてFAIL/blockedとする。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法・期待結果 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | net.Pipeでserverへ `GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\n\r\n` を一回だけ送り、valid Authorityのcertificate-only public PEM fresh copyを返すexact 200 wireをbyte-for-byte確認する。GET以外、path違い、query/fragment、Host等の追加header、header順/空白/Content-Length非canonical、body、header完了直後の追加byteは既存固定403とcloseになり、handler/control/Authorityを呼ばない。 | `focused-rerun` / strict request parserとone-operation境界はhermetic・deterministicである。 |
| QA-002 | AC-1, AC-2 | Authority nil、typed-nil、空/短小/4,097 byte以上、parse不能、private key block、CERTIFICATE以外、複数block、PEM後trailing byteを投入する。すべて200前に固定403となり、response・errorにPEM/private materialを含めない。 | `focused-rerun` / response前のAuthority distrustと非漏洩を直接再現できる。 |
| QA-003 | AC-2 | 有効な1〜4,096 byteの単一canonical CERTIFICATE PEMだけについて、self-signed ECDSA P-256、BasicConstraints CA、CertSign、現在時刻で有効を検証し、`HTTP/1.1 200 OK`、exact Content-Type、canonical Content-Length、`Connection: close`、body一回だけのwireを確認する。RSA/ECDSA P-384、非self-signed、BasicConstraintsなし/non-CA、CertSignなし、expired/not-yet-validを拒否する。 | `focused-rerun` / x509 fixtureはdeterministicに固定できる。 |
| QA-004 | AC-3 | fake dialer/net.Pipeでabsolute canonical socketだけを`unix`へ一回dialし、write/read deadline設定、partial writeの完走、必ずclose、200 response直後のEOFを確認する。相対/cleanでないsocket、dial nil/error、deadline/write/read/close failure、early EOFは固定context-free client errorとなる。 | `focused-rerun` / one-dial、deadline、closeを実OSなしで計測できる。 |
| QA-005 | AC-3 | clientがexact `GET /v1/proxy-ca HTTP/1.1` request（canonical zero Content-Length以外のheader/bodyなし）を送ること、accepted responseが唯一のbounded 200・固定status/header order/framing・canonical body length・Connection close・body後EOFのみであることをbyte-levelに確認する。chunked、header追加/並替/duplicate、leading-zero/length不一致、body不足/過剰、early/extra bytes、1xx/204/3xx/403/5xxは固定errorとなる。 | `focused-rerun` / parser独立性とフレーミング上限をhermeticに再現できる。 |
| QA-006 | AC-3, AC-4 | client response bodyへ、private key、multiple/noncanonical PEM、trailing byte、malformed parse、P-384/non-CA/CertSignなし/expired/not-yet-valid certificateを投入して固定errorを確認する。成功PEMはcallerが変更してもtransport/stateへaliasせず、後続取得値も変化しないfresh public copyである。error、stdout、stderrにPEM、subject、socket path、下位errorがないことをsentinelで確認する。 | `focused-rerun` / client側の独立CA validation、copy isolation、non-leakは固定fixtureで検出可能である。 |
| QA-007 | AC-1, AC-4 | candidate testとdiffを監査し、既存CONNECT/TLS、Issue/Revoke exact wire、credential helper get/store/eraseの期待値が維持され、GETがgeneric endpoint、cache/retry/fallback、file/environment参照、subject/query入力を導入しないことを確認する。 | `evidence-review` / 既存挙動全体への非干渉はcandidate diffと既存回帰証跡の独立監査が適切である。 |
| QA-008 | AC-5 | `candidate_commit`の親との差分を許可6パスだけへ限定し、計画目安約800〜1,100行に対する実績（追加・削除を明記した算術）と下振れ理由を監査する。既存実装の再利用による下振れは水増しせず許容する。dependency/Schema/Kakesu runtime/generated file/launcher/config/live state/実秘密がないこと、HANDOVERのfocused race、harness check/distcheck、candidate gate root check、diff checkのcommand/resultとcandidate gate invariantを監査する。 | `evidence-review` / full checksはcandidate-bound証跡の監査対象であり、raw log/digestを増やさずQA再実行は一回のfocused-raceだけに制限する。 |
| QA-009 | AC-4, AC-5 | 実OS Unix socketのowner/mode・別UID・peer binder、Git/libcurl/NSS trust、GitHub/OpenAI/DNS/TLS、systemd/VPS、実launcherのtrust-file作成・cleanup・Git configを実環境で確認する。 | `live-e2e` / TASKの範囲外かつ承認済み環境と安全なcleanupが未指定。現時点は `blocked/not-run` とし、他ケースPASSで代替しない。 |

## 境界・異常・回帰の判定

- QA-001〜006は上記単一 `go test -race` invocationに含める。各テスト名、fixture、期待wire、失敗時に機密値をassertionへ出さない方法をQA_RESULTへ記録する。race failure、timeout、漏洩、unexpected dial/close、fixed 403/errorからの逸脱はFAILである。
- PEM有効性の時刻依存を避けるため、テスト用clockまたは期限に十分余裕のある固定fixtureを用いる。実時刻に依存して結果が揺れるfixtureはfocused-rerun不適格でありFAIL/blockedに分類する。
- 403の正確な固定wireとclient errorの固定文字列はcandidateの既存定数に照らして監査する。原因別の詳細、PEM/subject/socket/path/下位errorを露出する差分はnon-leak違反（FAIL）である。
- `Authority` public accessorを共有parserでclientへ流用していないことを確認する。同一validator関数の共有でclient独立検証が失われる、又はprivate-key取得可能な広いAuthority APIを追加する場合は安全境界違反（FAIL）である。
- candidate diffに許可外パス、line budget逸脱、依存/Schema/config/生成物、実秘密、launcher又はlive-state mutationがあればscope failureとし、機能テストPASSでもQAはPASSしない。
- harness/root full checkの失敗は直ちにDEV faultと決めない。command/result記録、candidate一致、既知の環境依存性、再現性を確認し、`implementation defect`、`test/fixture defect`、`environment/infrastructure`、`evidence missing`、`out-of-scope live blocked` のいずれかに分類して記録する。

## 実装後の再確認

- [ ] HANDOVERの`candidate_commit`、親commit、QA実行tree、DEV証跡のcommitが一致することを確認した。
- [ ] QA-001〜006を一回のbounded hermetic `go test -race`で独立実行し、ケース別の結果を記録した。
- [ ] QA-007〜008のcandidate-bound evidence（test失敗検出能力、negative cases、弱体化の有無を含む）を独立監査した。
- [ ] QA-009をlive-e2eとして別記し、承認環境又は安全なcleanupがない限り`blocked/not-run`のままとした。
- [ ] 実装差分とレビュー結果を確認した。期待結果または範囲を変更する必要があれば、実行を停止してmain Agentの再承認を得る。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | | 初版 | `pending` |
| 2 | 2026-08-02 | qa-agent-terra-medium | TASK-first独立QA計画。AC1〜5、focused-race一回、candidate証跡監査、live blocked境界を固定 | `approved` |
| 3 | 2026-08-02 | main-agent-sol-high | 計画の約800〜1,100行は下限gateではなく目安であり、既存transport再利用による630 changed linesへの下振れを水増しせず受理すると明確化。 | `approved` |
| 4 | 2026-08-02 | main-agent-sol-high | 現行最小証跡契約に合わせ、raw exit log要求を削除しHANDOVER command/resultとcandidate gate invariantの監査へ補正。 | `approved` |
