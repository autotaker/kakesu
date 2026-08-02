---
task_id: "TASK-0063"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T01:20:43Z"
revision: 1
implementation_reviewed_at: "2026-08-02T01:50:45Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0063 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。同一candidateを、Git Smart HTTPの正規upload-pack readだけ、peer-derived subjectに束縛されたGit read capability、HTTP Basicのopaque handleから実tokenへの一回だけの置換、全層のfail-closed境界、既存REST/OpenAI契約の不変という観点で独立に評価する。

実GitHub、DNS/TLS、GitHub App token、別UID/NSS、systemd、VPSは本Taskのhermetic testでは確認しない。外部環境と安全なcleanupがないため、これらのlive-e2eは `blocked` のままにし、focused rerun又は証跡監査のPASSで代替・主張しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | config allowlist内の正規`owner/repo`に限り、GET `/<owner>/<repo>.git/info/refs?service=git-upload-pack` とPOST `/<owner>/<repo>.git/git-upload-pack`だけがGit readとして評価されることを確認する。method、path、単一canonical query、request media type、bounded bodyを厳密に検査し、receive-pack、extra/missing query、URL encoding、dot/empty segment、repository/host/operation mismatch、誤Content-Type、過剰又は不正bodyをcredential取得・network到達前に拒否することをfailure-detectする。 | `focused-rerun` / policyとrequest fixtureを閉じたhermetic・deterministic・bounded suiteで、正規許可と各拒否の到達前境界を再現できる。 |
| QA-002 | AC-2 | 明示的なGit read selectorだけがpeer-derived subjectへ、`provider=github`、同一repository、Git read operation、`github.com` host、TTL 5分、1回使用のCapabilityを発行することを確認する。caller入力によるsubject補完、allowlist外又はnoncanonical repository、無効selector/mismatchを発行前に拒否し、既存GitHub REST issue及びOpenAI issueのprovider/operation/allowlistの意味を変えないことを検出する。 | `focused-rerun` / clock、peer context、Registry、policy fakeに閉じたfixtureで、issue条件と既存scopeの回帰を決定的に検査できる。 |
| QA-003 | AC-3 | Git readだけが厳密なHTTP Basic `x-access-token:cap_...`を受理し、handleを同一Registryで一回消費してからresolverを一度だけ呼び、upstreamへ`x-access-token:<real token>`のBasicを一度だけ送ることを確認する。Bearer、malformed Basic、非`cap_...`、scope/subject/repository/operation mismatch、reuse、resolver失敗を拒否し、Agent入力のBasic又はhandleをupstreamへforwardしないことをfailure-detectする。response/error/diagnosticにhandle、real token、URL、下位errorがないことも確認する。 | `focused-rerun` / real Registryとcounting resolver/transportのcomposition fixtureにより、issue→consume→replacement→reuse rejectと非漏洩を外部credentialなしで観測できる。 |
| QA-004 | AC-4 | CONNECT、CA、inner HTTP mapping、policy、transaction、forwarder、pinned transportの各層が`github.com:443`を正規Git read flowだけへ渡し、host又はrequest形状の逸脱を次層・credential・network前に固定拒否することをsource/testとfixtureで確認する。redirect/retry/credential forwarding先変更がなく、discovery/postの成功statusとoperation対応のresponse media type、bounded binary response sizeを検査した後だけsinkへ渡し、unexpected status/media type/oversize/responseを拒否することをfailure-detectする。 | `focused-rerun` / injected connection、CA、mapper、resolver、transport、bounded response sinkを用いる一つのhermetic composition suiteで、全層の正規flowとnegative boundaryを再現できる。 |
| QA-005 | AC-5 | push/receive-pack、repository/host/operation/subject mismatch、malformed Basic、canonical URL逸脱、request media type/body逸脱、unexpected upstream responseの拒否を再確認する。同じcandidateで既存GitHub RESTとOpenAIが従来のBearer/token、JSON、CONNECT/CA/inner HTTP/policy/transaction/forwarder/pinned transport契約を維持し、既存testのskip、削除、弱体化、Git readへの過度な一般化がないことをfailure-detectする。 | `focused-rerun` / affected packageのrace suiteはhermetic・deterministic・boundedであり、Git read追加による既存REST/OpenAI回帰を同一candidateで検出できる。 |
| QA-006 | AC-6 | candidate diffが許可20 path以内でおおむね1,000〜1,400行であること、READMEが実環境を過大に保証しないことを確認する。helper/launcher/Approval/live state、新listener/socket/TCP入口、credential/token Agent側保存、registry persistence/cache/retry、Kakesu runtime、Schema、依存、生成物への差分がないことを監査する。DEVのfocused race、root `make check`、`git diff --check`と、Reviewerによるcandidate diff/root check監査がcandidate-boundのコマンド・cwd・結果を持つことを確認し、QAはroot checkを重複実行しない。 | `evidence-review` / candidate diff、README、focused test本文、DEV/Reviewerのcandidate-bound証跡で、scope・サイズ・検査実施・弱体化なしを独立監査できる。 |
| QA-007 | 対象外（AC-4/AC-6の環境依存境界） | 実GitHub Smart HTTP、実DNS/TLS、GitHub App token、別UID/NSS、systemd socket、VPS配置を確認する。 | `live-e2e` / `blocked`。承認済み実環境と安全なcleanupがTaskにない。QA-001〜006のPASSはこのケースをPASSにしない。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜005を次の一回だけ実行する。対象testは少なくとも、正規discovery/POST、receive-pack拒否、canonical URL/method/query/media type/body拒否、peer subjectに束縛されたGit read発行、same Registryのconsume/reuse reject、Basic handleからreal tokenへの一回置換、全層のhost/redirect/retry境界、binary response検査、REST/OpenAI回帰のいずれかを壊すと失敗しなければならない。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/egresspolicy ./internal/capability ./internal/capabilitycontrol ./internal/connectsession ./internal/proxyca ./internal/brokerhttp ./internal/egresstransaction ./internal/upstreamforwarder ./internal/upstreamtransport
```

zero exitだけでは不十分である。candidate不一致、test欠落/skip/弱体化、外部GitHub・DNS・実tokenへの依存、unbounded又はnon-deterministic fixture、必要なnegative assertionの欠落は該当ケースをPASSにしない。root `make check`はDEV一回とReviewerの証跡監査に限り、QAはこのfocused Go race suite以外を再実行しない。

## 境界・異常・回帰

- Git readはGitHub RESTとprovider名だけで同一視しない。operation、host、repository、peer-derived subject、authorization scheme、request/response media typeを全層で完全一致させ、upload-pack以外を許可しない。
- discovery queryはcanonicalな単一`service=git-upload-pack`だけである。encoding、dot/empty segment、余分/欠落query、redirect、retry、forward先変更、Git LFS/submodule/archive、GitHub Web/GraphQL/REST writeは拒否又は対象外のままとする。
- capabilityは5分・一回使用で同一Registryだけから消費する。Basic credentialのhandle抽出、consume、resolver、real token置換の順序を固定し、Agent由来credential、handle、real tokenを保存、再利用、forward、診断表示しない。
- Git binary responseはJSONとして扱わず、operation対応media type、成功status、size上限の検査後だけsinkへ渡す。unexpected response、下位error、URL、credentialを診断又はresponseへ含めない。
- external live境界は `blocked` として残す。focused failureはcandidate、test fixture、実行環境、要件又は証跡に分類し、DEV不具合とは決めつけない。

## 実装後の再確認

- [x] HANDOVERの`candidate_commit`を基に、同一candidateをQA-001〜006で独立に評価した。
- [x] QA-001〜005のfocused Go race suiteを一回だけ実行し、正規Git read、push拒否、strict negative boundaries、Basic credential replacement、binary response検査、REST/OpenAI回帰のfailure detectionを確認した。
- [x] QA-006のcandidate diff、許可20 path、行数、対象外差分、DEV `make check`/`git diff --check`、Reviewer証跡を独立監査した。
- [x] QA-007をunit PASSと分離して `blocked` のまま記録した。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。Git Smart HTTP upload-pack限定、peer-bound capability、Basic credential置換、全層strict boundary、push拒否、REST/OpenAI回帰、非漏洩とlive境界を定義。 | `approved` |
