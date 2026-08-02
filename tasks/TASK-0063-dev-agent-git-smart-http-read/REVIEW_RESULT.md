---
task_id: "TASK-0063"
status: complete
reviewer_agent: "reviewer-agent-terra-medium"
decision: pass
reviewed_at: "2026-08-02T01:50:58Z"
---

# TASK-0063 REVIEW RESULT

## 監査対象

- HANDOVERに固定された新candidateだけを、planning commitからの差分として独立再監査した。candidateの識別子はHANDOVERを正本とし、本結果へ複製しない。
- TASK.md、承認済みPLAN.md、HANDOVER.md、20許可パスのsource/test、および適用AGENTS.mdを照合した。QA_RESULTは読まず、QAの結果を待たずに判定した。

## 監査したDEV証跡

| コマンド/テスト | 結果 | 備考 |
|---|---|---|
| DEV root `make check` | `PASS（証跡監査）` | HANDOVERの再固定candidate-bound最終PASSを確認した。指示に従いroot checkを重複実行していない。 |
| affected 9 package `go test -race` | `PASS（証跡監査）` | HANDOVER記録と、Git policy/control/transaction/CONNECT/CA/handler/forwarder/transportのnegative testsを照合した。 |
| planning commitからのcandidate diff | `PASS` | 許可された20パスのみ、1,008 additions / 79 deletions。helper/launcher/Approval、listener/socket、依存、Schema、Kakesu runtime、生成物、live stateは含まれない。 |
| `git diff --check` | `PASS` | planning commitからのcandidate diffに空白エラーなし。 |

## 受け入れ条件の確認

| 条件 | 結果 | 根拠 |
|---|---|---|
| AC-1 | `pass` | allowlist内のdiscovery/upload-packだけを厳密に許可し、`:443`を含むdefault HTTPS authorityはpinned transportでcanonical hostnameへ安全に正規化される。receive-pack、method/query/media/body/URL逸脱はcredential・network到達前に拒否される。 |
| AC-2 | `pass` | 明示`github-git-read` selectorだけがpeer-derived subject、allowlist repository、固定5分・1回のGit scopeを発行する。REST selector省略とOpenAI selectorの既存意味も維持される。 |
| AC-3 | `pass` | Git scopeだけがcanonical HTTP Basicの`x-access-token:cap_...`を受理する。consume後にresolverを一回呼び、prepared requestにはreal-tokenのBasicだけを渡す。malformed Basic、重複Authorization、resolver failure、reuseのnegative testsがある。 |
| AC-4 | `pass` | CONNECT、CA、inner mapping、forwarder、pinned transportのhost/operation/media checksを確認した。`:443` positive testはDNS resolverをport-free `github.com`で一回呼び、IP literal `:443` dial、`github.com` SNI、HTTP/1.1を観測する。redirect/retryは追加されていない。 |
| AC-5 | `pass` | receive-pack、query/URL/media/body逸脱、subject/scope mismatch、malformed Basic、unexpected responseを拒否するtestを確認した。REST/OpenAIのBearer/JSON/transport testsは残り、削除・緩和はない。 |
| AC-6 | `pass` | 20パス・行数・対象外差分は計画上限内で、DEVのfocused race、再固定root `make check`、`git diff --check`のPASS証跡がある。 |

## 指摘

| ID | 重大度 | 状態 | 内容 | 根拠 |
|---|---|---|---|---|
| F-1 | high | resolved | 旧candidateではpolicy/inner requestが許可する`github.com:443`をpinned transportが拒否した。新candidateは各許可authorityのhost又はhost`:443`だけを受理し、戻り値をport-free canonical hostnameへ固定する。`github.com:443` positive testでDNS resolver、IP literal dial、SNIまでを確認し、`:444`、near-match、URL Host mismatchはresolver前に拒否する。 | `upstreamtransport.go:204-235`、`upstreamtransport_test.go` の全host default-port、Git positive、near-match/`:444` test。 |

## 残存リスク

- 実GitHub credential/DNS/TLS、実Git client、別UID/NSS、systemd/VPSは承認済みlive E2E環境がないため未確認であり、hermetic証跡はそれらを代替しない。

## 結論

`pass` — 新candidateは承認済みPLANの20パス制限とAC-1〜AC-6に適合する。旧F-1はcanonical default-portの厳密な受理・normalizationとpositive/negative transport testsで解消された。
