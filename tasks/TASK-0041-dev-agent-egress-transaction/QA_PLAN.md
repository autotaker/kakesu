---
task_id: "TASK-0041"
title: "Capability連携済みegress transaction QA plan"
status: approved
qa_agent: qa-agent-terra-medium
approved_by: main-agent-sol-high
approved_at: "2026-08-01T06:12:29Z"
implementation_reviewed_at: "2026-08-01T06:27:47Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# QA_PLAN: TASK-0041

## QA scope

Candidate の `tools/dev-agent-harness/internal/egresspolicy/`、
`tools/dev-agent-harness/internal/egresstransaction/`、および
`tools/dev-agent-harness/README.md` の差分を、TASK.md の受け入れ条件に対して独立に確認する。

実Credentialのファイル読取・保存・生成・更新、network/TLS/HTTP forwarding、server/proxy、
DNS、upstream通信は対象外であり、これらについてPASSを主張しない。

## Candidate binding

QAはHANDOVERに記録された `candidate_commit` を唯一のcandidateとして扱う。候補が再固定された場合は、
本QA_PLANに基づく確認をそのcandidateへやり直す。

## Cases

| Case ID | 対象AC | 確認内容 | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | allowlistの同一評価からGitHub REST readとOpenAI Responses textのcanonical scopeが導出され、denyではzero scopeと既存固定`request-denied`となること、既存`Authorize`のdecision/error互換性をcandidateのテストと差分で監査する。 | evidence-review | candidate / candidate差分とDEV command/result |
| QA-002 | AC-2 | policy、capability Registry、CredentialResolver、ForwarderのnilとCredential最大長境界が固定Rules errorとなり、non-nil zero policy/RegistryはExecute時に固定denyとなること、canonical subject、caller-owned Authorization/body sliceの入力不変をcandidateのテストと差分で監査する。 | evidence-review | candidate / candidate差分とDEV command/result |
| QA-003 | AC-3 | policy allow、provider別Authorization抽出、scope API由来の完全一致Consume、resolver/Forwarder非到達の順序をcandidateのテストと差分で監査する。入力の大小文字、余分な空白、複数値、改行、別schemeの拒否を含む。 | evidence-review | candidate / candidate差分とDEV command/result |
| QA-004 | AC-4 | Consume成功後のみcanonical provider/repositoryでresolverを一回呼ぶこと、resolver/Forwarder失敗時の消費済みfail-closed、Credentialの空・長さ・文字種拒否、valid CredentialだけをForwarderへ一度渡すこと、並行1-useのexactly-one到達を確認する。 | focused-rerun | candidate / `cd tools/dev-agent-harness && GOCACHE=/tmp/task-0041-qa-gocache go test -race ./internal/egresspolicy ./internal/egresstransaction` / result |
| QA-005 | AC-5 | Forwarderへ同期的に渡るPreparedRequestが独立copyのmethod/raw URL/content type/body、canonical provider scope、上流`Bearer` Credentialのみを持ち、入力Authorization/opaque handleを保持しないこと、ExecuteがCredential-bearing値を返さずTransactionが保持しないこと、固定non-leak errorをcandidateのテストと差分で監査する。 | evidence-review | candidate / candidate差分とDEV command/result |
| QA-006 | AC-6 | table-driven unit coverageがscope導出、Authorize互換、両provider成功、Authorization境界、各deny、Credential検証、消費順序、Forwarder handoff、入力不変、non-leak、並行1-useを検出することを監査する。root `make check` と harness `make check`/`make distcheck` はDEV証跡の監査のみとし、QAは再実行しない。candidateの許可pathと、base...candidateの`git diff --numstat`追加＋削除合計が1,200以下であることもDEV証跡とcandidate差分で確認する。 | evidence-review | candidate / candidate差分とDEV command/result |

## Execution rule

`focused-rerun` はQA-004のため、指定されたrace test commandをcandidateで一度だけ実行する。ほかのケースはcandidate-boundなDEV証跡とcandidate差分を独立監査する。root `make check`、harness `make check`、harness `make distcheck`は再実行しない。

## Result criteria

各caseの証跡はcase ID、candidate、command、resultだけを記録する。対象外の実Credential、file、network、TLS、HTTP forwardingに関する確認不足を他モードのPASSで代替しない。失敗はQAガイドラインに従い、実装不具合と決めつけずに分類する。

## 実装後の再確認

- [x] fixed candidateの実装差分と、REVIEWを開始条件としない独立QA結果を確認した。
- [x] QA-004のfocused race testをcandidateで一度だけ実行し、source/test auditと合わせて確認した。
- [x] QA-001〜006のcandidate-bound証跡、許可5 path、差分692行、DEV command/resultを監査し、期待結果または範囲を変更していない。

## Approval

承認者: main-agent-sol-high

承認日時: 2026-08-01T06:12:29Z
