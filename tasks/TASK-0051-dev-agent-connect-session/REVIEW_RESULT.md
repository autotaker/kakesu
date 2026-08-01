---
task_id: "TASK-0051"
status: reviewed
reviewer_agent: "reviewer-agent-terra-medium"
decision: pass
reviewed_at: "2026-08-01T13:51:40Z"
---

# TASK-0051 REVIEW RESULT

## 監査範囲と方法

- main管理の最新`TASK.md`（revision 2のplanning input packet）、承認済み`PLAN.md`、`QA_PLAN.md`、`HANDOVER.md`を正本として、HANDOVERに固定された新candidateの全製品差分と関連する既存`brokerhttp.Handler`を独立にsource/evidence reviewした。
- テスト、`make`、lintその他の実行コマンドは実施していない。HANDOVERのDEV実行記録はcandidate-bound証跡としてのみ確認した。
- candidate worktreeのHEADはHANDOVERに固定されたcandidateと一致し、作業ツリーはcleanだった。planning baseとの差分は許可された3 fileのみ（README、`internal/connectsession/session.go`、`session_test.go`）、`+1000/-0`。`git diff --check`に出力はなかった。

## 修正後契約の確認

注入Handlerは`brokerhttp.Handler`相当のtrusted dependencyであり、request contextのdeadline/cancelへ協調してreturnする。Sessionの保証はconnection I/O deadline、同一request contextの伝播、自身のcancel watcherおよびconnection cleanupまでであり、任意のnon-cooperative callbackの強制停止は対象外である。`session.go`はHandlerを同期的に呼び、別goroutineに包んだtimeoutによるleakを導入していない。従って、先行レビューR1が要求していたnon-cooperative Handlerの強制停止は、revision 2のAC-5/AC-6およびQA_PLANの依存境界には適用されない。

## 先行指摘の再監査

| 項目 | 結果 | 根拠 |
|---|---|---|
| 1 MiB response body cap / persistent failure / no partial output | pass | `Write`は上限超過時に`failed`を永続化し、handlerがerrorを無視しても`flush`はwire出力前に失敗する。全header/bodyを検証・bufferしてから一回のserializationへ進み、overflow負テストはinner responseなしとcloseを検出する。 |
| response header token/value validationとcase-variant reserved framing | pass | `validResponseHeaders`はRFC tokenのheader name、visible-ASCII value、canonical map keyをclone/serialize前に検証する。non-canonical `content-length`は拒否され、canonical `Transfer-Encoding`は固定framing設定前に除去される。 |
| pre-clone 16 KiB aggregate header budget | pass | fixed `Content-Length`/`Connection`とheader終端を先にbudgetへ含め、検証済みheaderだけをcopyする。aggregate-header負テストはwire outputなしを確認する。 |
| strict CONNECT / fixed response | pass | 最大16 KiBのraw header、exact two-host `:443` authority/Host、唯一のoptional `User-Agent`/`Proxy-Connection: keep-alive`以外を依存前に拒否する。fixed empty 403と、Issue成功後だけのfixed empty 200をsource/testで確認した。 |
| TLS / context / close ownership | pass | Issueは200前に一回だけ呼ばれ、TLS 1.2+、exact SNI、ALPN `http/1.1`を確認する。caller contextからphase deadlineを導出してrequestへ渡し、cancel watcherとdeferでSession所有connをcloseする。cooperative cancel、panic、CONNECT stall、TLS failure、並行session隔離をtest sourceが検出する。 |
| HTTP phase deadline propagation regression | pass | 新candidateの`TestHTTPPhaseDeadlinePropagation`はHandler内でrequest context deadlineを観測する。callerの2秒deadlineは許容誤差内でそのまま伝播し、caller deadlineなしではdeadlineが存在し、観測時点から5秒capを超えないことを検出する。`newPhaseContext`と`setDeadline`は同じHTTP phase contextをconn I/Oとrequestに使う。 |

## 受け入れ条件の確認

| 条件 | 結果 | 根拠 |
|---|---|---|
| AC-1 | pass | `New`はtyped-nilを含むAuthority/Handlerを拒否し、Sessionはその二依存だけを保持する。nil/zero/破損Session、nil conn/contextは固定errorへ畳み、`Format`は依存詳細を出さない。 |
| AC-2 | pass | raw bounded parserとstrict validationがmethod/version/authority、重複・未知・framing・auth・upgrade header、過長/control User-Agent、early byteをAuthority/Handler前にfail closedする。test sourceはfixed 403、dependency未到達を確認する。 |
| AC-3 | pass | `Issue(target)`は一回かつ200前で、issued certificateだけをTLS serverへ渡す。TLS version、SNI、ALPNまたはhandshake失敗は追加HTTP response、retry、fallbackなしでcloseする。 |
| AC-4 | pass | TLS後のHTTP/1.1 requestはcaller由来のphase contextを`WithContext`で継承してinjected Handlerへ同期一回だけ渡る。real `brokerhttp.Handler` fixtureはcontext-only resolverとExchange単回を確認し、outer CONNECT/RemoteAddr/SNIから主体を作るコードはない。 |
| AC-5 | pass | CONNECT/TLS/HTTP I/Oへphase deadlineまたは早いcaller deadlineを設定し、caller cancellationはwatcherがconnをcloseする。HTTP Handlerにも同一phase deadlineが伝播することを新しいcaller-earlier/5秒-cap回帰testで確認する。panic recovery、defer close、協調的cancel Handler returnをsource/testで確認した。non-cooperative callbackの強制停止は修正後のtrusted dependency契約外であり、要求していない。 |
| AC-6 | pass | hermetic `net.Pipe`/in-memory CA/real Handler fixtureのtest sourceはtwo-host success、fixed responses、strict rejection、Issue単回、TLS negative、context/identity、Handler単回、HTTP phase deadline propagation、cooperative cancel、panic、stall、close、parallel isolationとserializer negative casesを失敗検出する。HANDOVERはcandidate-bound race test、harness/root check（candidate launcherを含む）、distcheck、README lint、diff checkを`PASS`と記録する。 |

## live-e2e境界

実listener bind/accept、OS peer identity、CA file/trust lifecycle、real client、gh/OpenAI、network namespace/VPSは未実装・未確認である。QA-007の`live-e2e — blocked`をhermetic source/evidenceのpassで置換していない。

## 指摘

なし。

## 結論

`pass` — candidateは修正後のtrusted cooperative Handler契約とAC-1〜AC-6に適合する。先行R1は、revision 2で明文化された依存境界の外側にnon-cooperative callbackの強制停止を要求していたため、remaining findingではない。
