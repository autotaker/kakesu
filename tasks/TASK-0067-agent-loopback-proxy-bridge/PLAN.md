---
task_id: "TASK-0067"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Agent が接続できる TCP endpoint と peer-bound Unix egress の間に新しい接続ライフサイクルを置き、loopback 固定、bounded concurrency、half-close、cancel/drain を同時に fail-closed に保つ必要がある。"
approved_dev_profile_risk_signals:
  - "Agent namespace 内の TCP-to-Unix egress boundary"
  - "trusted fixed Unix socket path と caller-controlled input の分離"
  - "concurrent bidirectional stream、half-close、cancellation、FD/goroutine cleanup"
  - "公開 error と connection-local failure の non-leak boundary"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T03:50:39Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T03:50:39Z"
classification_approval_reason: "Agent向けTCP endpointとUnix egress間の外部観測可能な接続ライフサイクルを追加する製品変更。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Agent loopback proxy bridge

## 根拠と分類

唯一の要求根拠は `TASK.md` の `Planning input packet` である。Agent が利用する TCP HTTP-proxy endpoint を新設し、既存の peer-bound egress Unix socket までの接続受付・dial・stream 終了を外部観測可能な製品境界として追加するため、`change_class` は `product` とする。DEV は packet 指定の `sol-high` とする。本PLANには独立PLANレビューを設けず、Main が TASK-first で独立作成される `QA_PLAN.md` とともに意図、scope、受け入れ経路を確認して承認を記録する。

candidate は packet の3許可パスだけを変更し、追加・削除を合わせて約650〜950行に収める。bridge は `internal/proxybridge` 内に閉じ、launcher/command、子 process、signal、environment allowlist、temporary directory、CA trust file、Git configuration、credential helper、設定、Makefile、依存、Schema、Kakesu runtime、生成物、live state を変更しない。既存 egress Unix socket の peer binding、CONNECT/control 認可、TLS/inner HTTP policy、credential replacement、DNS/upstream の意味も変更しない。

listen address/port、egress socket path、TCP upstream、environment override を API、CLI、environment、設定のいずれからも受け取らない。HTTP/CONNECT/TLSを parse せず、wildcard/IPv6/fixed port、TCP 又は別 Unix socket、retry/fallback/cache、audit/log、実秘密を追加しない。実OS namespace、loopback 到達性、Unix socket permission/peer UID、実 Git/gh/OpenAI、systemd/VPS は hermetic PASS の代替対象にしない。

## end-to-end設計

1. `proxybridge` は trusted constructor に固定値だけを受け入れる小さな lifecycle boundary とする。公開 `Rules` は固定済み absolute/clean Unix socket path と、1〜64 の `MaxConcurrent` だけを表現し、constructor は空、relative、clean でない Unix path と上限外を固定 `ErrInvalidRules` に畳む。実装で選ぶ listener は `tcp4` と正確な `127.0.0.1:0` に一回だけ限定し、OS が返した `*net.TCPAddr` が IPv4 loopback、非zero の ephemeral port であることを再検証してからだけ server を返す。
   - listener creation は package-private seam に閉じ、production は `net.Listen("tcp4", "127.0.0.1:0")` 以外を呼ばない。test double の address も constructor/startup 完了前に同じ shape を検証する。nil interface と typed-nil の listener/dialer は reflect を使う既存 listener package と同様に区別して拒否し、listen成功後に不正な address 又は listener を返した seam の non-nil listener は panic guard 下で一回 Close する。
   - 成功 API は caller が子 client へ渡す canonical endpoint だけを返す。URLは `http://127.0.0.1:<decimal port>` をその固定 form で組み立て、host alias、bracket、IPv6、path/query/userinfo、untrusted socket information を含めない。Unix path は `Server` にだけ保持し、endpoint や public error の formatting に出さない。

2. `Serve(ctx)` が one-listener run の所有者となる。開始時入力、既に cancelled context、listener の nil/typed-nil、又は internal state 不整合は accept 前の固定 error にする。正常 run では cancellation watcher が listener を一度だけ閉じ、accept loop は slot を取得してから `Accept` するため、上限中に余分な client を accept して Unix dial を開始することはない。
   - `Accept` が connection なしで失敗、nil/typed-nil connection、又は予期しない panic を返した場合、new accepts を永久に停止し、run context を cancel、listener を close、active connection を drain して固定 `ErrServer` を返す。親 context 由来の listener close は server failure ではなく正常終了とする。Close は `sync.Once` と panic guard で一度だけ所有し、WaitGroup の Add を goroutine 起動前、Wait を accept loop 停止後に行う。
   - `Serve` は retry、listener replacement、alternate endpoint を作らない。concurrency 上限は request/byte size/session duration の proxy policy ではなく、dial 前の connection slot だけに適用する。active stream に独自 deadline を導入せず、親 cancellation、EOF、I/O error を唯一の終了条件に残す。

3. accepted client 一件は固定 Unix socket へ一回だけ接続する。connection goroutine は run context から child context を作り、`net.Dialer.DialContext(ctx, "unix", fixedPath)` の一回だけの呼出しを、package-private dial seam で test 可能にする。dial phase だけに bounded timeout を設け、timeout、cancel、nil/typed-nil conn、dial error は client を close して終了する connection-local denial とする。
   - dial failure では client payload を read/forward せず、retry、別 network/path、TCP fallback、diagnostic response を出さない。公開 `Serve` error は accepted connection 一件の dial/copy/close failure では変えず、socket path、client address、lower error を保持しない。
   - dial success 後にだけ、raw client と Unix upstream を対にして stream を始める。bridge は bytes、HTTP header、TLS record、credential を inspect、buffer、rewrite、log しない。既存 `connectsession` が downstream protocol/authorization の唯一の正本であり、bridge からそこへ identity/authority を生成又は移送しない。

4. stream は two directional copy と explicit half-close を connection lifecycle 内に閉じる。一方向の `io.Copy` が EOF で成功したときだけ、反対端が `CloseWrite` を提供すれば一回呼び、相手からの応答を継続して読めるようにする。non-EOF copy failure、half-close failure、context cancellation は run child context の cancel と両端 Close を起動し、もう一方の blocked copy を解除する。
   - `net.Conn` は `CloseWrite` を必須 interface と仮定しない。production TCP/Unix connection と test wrapper に対して optional half-close interface を確認し、未対応の test double は full close によって blocked copy を解放する。client EOF と upstream EOF の各 direction、片方向 write failure、dial success後の cancel、Close failure が WaitGroup 完了を妨げないことを保証する。
   - connection-local panic は recover して close/release へ進み、accept loop や他 connection を server error に拡大しない。listener accept failureだけが active stream を一括 cancel/drain する server-level failure boundary である。どの failure も input、Unix path、address、下位 error を返す/format する形にはしない。

5. README は bridge を、後続 launcher が固定 Unix egress socket を HTTP-proxy-only Agent client へ提示するための前段としてだけ記載する。固定 IPv4 loopback/ephemeral endpoint、one Unix dial、bounded concurrency、raw bidirectional bytes と close ownership、egress service に残る authorization を明示する。
   - bridge が listener の namespace isolation、OS permission/peer UID、live client proxy support、CA trust、child-process lifecycle、HTTP policy を証明又は構成しないことを明記する。socket path、port、credentials、実 deployment/example configuration は文書にも出さない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | fixed `tcp4`/`127.0.0.1:0` listener seam と returned `TCPAddr` の二重検証で、一つの canonical endpoint 以外を表現不能にする。 | `internal/proxybridge/bridge.go`、`bridge_test.go` | 1 | invalid rules、nil/typed-nil、listen/address/port shape 逸脱は accept 前の fixed rejection。endpoint、error に input/path/address を出さない。 |
| AC-2 | trusted fixed Unix path を immutable server state に閉じ、per-client one dial + dial timeout を package-private seam で実装する。 | `internal/proxybridge/bridge.go`、`bridge_test.go` | 2 | dial/cancel/nil failure はその client だけを close。payload forwarding、retry、別 socket/network/fallback、lower-error diagnostic は行わない。 |
| AC-3 | paired copy、EOF 時だけの opposite `CloseWrite`、cancel/full-close fallback、WaitGroup drain を一 connection owner が扱う。 | `internal/proxybridge/bridge.go`、`bridge_test.go` | 3 | EOF は half-close 後 opposite direction を継続し、I/O/cancel/close failure は両端を閉じて goroutine/FD を回収する。HTTP/TLS は解釈しない。 |
| AC-4 | slot-before-Accept、one-shot listener close、accept failure と parent cancel の別 result、active drain を固定する。 | `internal/proxybridge/bridge.go`、`bridge_test.go` | 1、4 | upper bound 中は追加 Unix dial を始めない。unexpected accept failure は fixed server error + drain、parent cancel は正常終了 + drain。 |
| AC-5 | package source/test と README の3 path に scope を閉じ、fake listener/dialer と `net.Pipe` を使う race-focused evidence を candidate に結ぶ。 | 許可済み3パス | 1–5 | scope/line budget/検査逸脱は candidate に含めず Main へ戻す。live namespace、permission、real client の確認は hermetic PASS に読み替えない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/proxybridge/bridge.go` | fixed loopback listener creation/validation、canonical endpoint、trusted Unix-path retention、bounded one-run Serve、single timed Unix dial、bidirectional stream/half-close、cancel/drain、fixed errorsとprivate seamsを実装する。 |
| `tools/dev-agent-harness/internal/proxybridge/bridge_test.go` | fake listener/dialer、tracked connections、`net.Pipe` を使い、endpoint/rules reject、dial ordering/count、raw stream/half-close、concurrency、accept/cancel/failure drain、non-leakを race-safe に確認する。 |
| `tools/dev-agent-harness/README.md` | loopback bridge と egress authorization の責務分離、後続 launcher との境界、hermetic coverage が保証しない live 条件を記載する。 |

## 実装手順

1. `bridge.go` に public minimal API と fixed errors を定義する。constructor で Unix path と concurrency を検証し、production listener seam を exact `tcp4` loopback/ephemeral の一回だけに固定する。returned listener address を型・IP・port まで検証して canonical endpoint を生成し、失敗なら listener を close して固定 error を返す。
2. listener、dialer、connection の typed-nil 判定と package-private test seams を追加する。seam は constructor/run 内で一度だけ使用し、export、configuration、environment override に昇格させない。endpoint/public error/format が fixed data だけであることを先にテストする。
3. slot-before-Accept の Serve loop と cancellation watcher を実装する。listener close を one-shot にし、unexpected accept failure と parent cancellation の result を分ける。accepted connection の WaitGroup ownership、slot release、panic-safe close、accept loop 停止後の drain を先に完成させる。
4. per-connection single Unix dial と raw stream lifecycle を接続する。dial timeout を phase に限定し、dial failureで client closeする。two copy workers、EOF half-close、non-EOF/cancel 時の full close、worker completion待機を実装し、client/upstream の全 close path で slot と goroutine が回収されるようにする。
5. test matrix を追加してから README を実装済み contract と一致させる。テストは production network、Unix filesystem、external service を開かず、fake listener/dialer と `net.Pipe` の bounded synchronization で完結させる。最後に3 path、行数目安、対象外差分を確認する。

## 検証計画

DEV の focused race suite は次の failure-detecting cases を持つ。

- `New`/start: empty/relative/unclean Unix path、0/65 concurrency、nil/typed-nil listener/dialer、listen error、wildcard/IPv6/non-loopback/fixed/zero port address、malformed listener addressを reject し、valid listener だけが fixed endpoint を返すこと。
- dial: accepted client ごとに fixed path + `unix` で一回だけ dial されること、concurrency slot 中には second dial/Accept が進まないこと、dial timeout/error/nil conn は client close と no bytes forwarded になること、failure value に path/address/lower error がないこと。
- stream: client→upstream と upstream→client の opaque byte round-trip、client EOF/upstream EOF の opposite write half-close、片方向 copy/half-close failure、context cancel、connection Close failureで両端・goroutine・slotが回収されること。HTTP/CONNECT/TLS decodeを要求する test は置かない。
- serve: unexpected accept error/nil conn が listener close、active cancellation/drain、fixed server errorとなること、already-cancelled/parent-cancel は accept を開始せず又は正常 close/drainすること、connection-local dial/copy failure は sibling stream を止めないこと。
- regression/scope: constructor/private seam の mutationが public APIへ新しい address/path/override を生まないこと、README が implementation boundaryを記述し、candidate diff が許可3パス外に出ないこと。

candidate 固定前に DEV は少なくとも次を実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/proxybridge
cd tools/dev-agent-harness && make check
make task-check TASK=TASK-0067
git diff --check
```

candidate gate は同じ固定 working bytes に対して root `make check` を一回実行する。QA_PLAN は AC ごとに `focused-rerun`、`evidence-review`、必要な `live-e2e` を理由付きで割り当てる。loopback exposure、trusted socket path、dial ordering、half-close/cancel/drain、non-leak は negative test と race evidence を独立監査し、根拠不足なら `evidence-review` PASS にしない。

実OS namespace/loopback 到達性、Unix socket permission/peer UID、real Git/gh/OpenAI proxy behavior、CA trust、launcher cleanup、systemd/VPS は `live-e2e` であり、承認済み環境と安全な cleanup がなければ blocked のまま残す。hermetic test はこれらの PASS を代替しない。

## リスクと復旧

- loopback binding が広がるリスクは、production listen string の固定、`tcp4`、returned `TCPAddr` の再検証、wildcard/IPv6/non-loopback/fixed-port fake matrixで抑える。
- Agent が socket/network/path を差し替えるリスクは、constructor内の validated fixed path と unexported dial seamだけにし、API/CLI/config/environment input を追加しないことで抑える。
- capacity を超えた接続が Unix egress へ届くリスクは、slot を Accept 前に取り、dial count と blocked second Accept を直接確認することで抑える。
- half-close が応答を切る、又は cancel/I/O failure が goroutine/FD を残すリスクは、direction別 `net.Pipe` EOF、tracked Close、failure injection、timeout-bounded Wait、`-race`で抑える。
- bridge が authorization proxy 又は diagnostic oracle になるリスクは、opaque byte copy、connection-local fail-close、fixed errors、path/address/lower-error non-leak assertionsで抑える。

復旧時は許可3パスの candidate 製品差分だけを戻し、Agent向け TCP endpoint を存在しない状態へ復元する。bridge は files、credentials、CA trust、configuration、external network state を作らないため、追加 cleanup はない。復旧後は focused race suite、harness/root `make check`、`make task-check TASK=TASK-0067`、`git diff --check` を再実行する。

## 引き継ぎ条件

DEV は Main 承認済み PLAN と独立 QA_PLAN の後、許可3パスだけで同一 candidate を一回固定する。Reviewer と QA は同一 candidate を相互の PASS を待たず独立に評価する。Main だけが stage/commit/merge/push、completion gate、`--no-ff --no-commit` 検査、main 統合、必要な環境依存確認を所有する。

candidate に launcher/process/environment、fixed socket path を選ばせる API/configuration、HTTP/TLS parsing、authorization policy、TCP fallback/retry/cache/log、dependency/Schema/generated/live state、実秘密を含めない。dependency-ready reconciliation は packet で N/A と固定済みであり、TASK-0051/0066 の peer-bound egress socket と strict control/CONNECT ownershipを変更しない。

## 未解決事項

- なし。packet が固定する Unix egress boundary と後続 launcher 分離を前提にする。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] fixed loopback endpoint、single Unix dial、slot-before-Accept、half-close/cancel/drain、existing egress authorization不変を具体化している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
