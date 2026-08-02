---
task_id: "TASK-0061"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T00:40:39Z"
revision: 1
implementation_reviewed_at: "2026-08-02T01:03:56Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0061 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。candidateの許可8 path内で、peer context由来subject、限定control protocol、既存Registry/transactionとの同一instance共有、revoke、固定非漏洩診断、既存CONNECT/TLS/HTTP/policy境界の不変を評価する。

実GitHub/OpenAI、DNS/TLS、NSS/別UID、systemd socket、VPSはこのTaskで外部状態を変更しない対象外境界であり、live-e2eを割り当てない。hermetic testのPASSをそれらの実環境事実の証拠として扱わない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | issuerがcaller入力ではなくpeer binderのcontextに束縛済みAgent instance/UID/workspaceだけをsubjectとして使うことを確認する。GitHubはconfig allowlist内の正規`owner/repo`一件だけ、OpenAIはconfigに少なくとも一つの許可modelがある場合だけ、固定TTL/usesの`cap_...`を発行する。missing/corrupt peer context、subject自己申告、allowlist外/noncanonical repository、OpenAI model list empty、無効provider/operationを発行前固定拒否として検出する。model個別値は既存policy検査へ委譲し、controlが独自allowlist規則を再実装しないことを確認する。 | `focused-rerun` / peer-contextとconfig/policyのfakeを用い、subject/allowlist/empty-model gateをhermetic・deterministic・boundedに再現できる。 |
| QA-002 | AC-2 | control protocolが既存egress socket上で通常HTTPS `CONNECT`と明確に分離され、一接続一操作、strict request、Content-Length必須、bounded bodyだけを受理してcloseすることを確認する。malformed/unknown method/path、over-limit、early bytes、chunked、upgrade、keep-alive、複数操作をcredential/issuer到達前に固定拒否し、CONNECT/TLS/HTTPの既存parser・timeout・close契約を弱めないことをfailure-detectする。 | `focused-rerun` / `net.Pipe`または同等in-memory listenerとcounting issuer/transportで、protocol分離と到達前拒否を外部networkなしに観測できる。 |
| QA-003 | AC-3 | service compositionがcontrol issuerと既存egresstransactionへ一つのin-memory `capability.Registry` instanceを渡すことを確認する。peer-derived subjectでissueしたhandleが対応GitHub/OpenAI scopeの既存transactionで一度だけconsumeされ、同じhandleのreuse、provider/repository/operation/subject mismatchは拒否されることを確認する。別registry、rollback、retry、cache、credential copy、発行後の失敗での再発行をnegative caseで検出する。 | `focused-rerun` / real Registryとfake resolver/transportのcomposition fixtureでissue→existing transaction consume→reuse rejectを完全に再現できる。 |
| QA-004 | AC-4 | revokeが正規handle一件を同じpeer-derived subjectに限り失効し、以後consumeを拒否することを確認する。unknown/malformed/expired handle、別instance/UID/workspace、不正provider/repository/operationでrevoke又はcredential取得へ進まないこと、raw handle、credential、URL、config、lower errorをerror/response/logへ出さないことをfailure-detectする。 | `focused-rerun` / clock/Registry/fake dependencyに閉じたfixtureでsubject-bound revoke、unknown/malformed、non-leakを決定的に確認できる。 |
| QA-005 | AC-5 | focused suiteがQA-001〜004のpeer subject、bounded control/CONNECT分離、allowlist/model gate、same Registry lifecycle、revoke、fixed diagnosticsを実際に失敗検出し、race-freeであることを確認する。既存CONNECT/TLS/HTTP、GitHub REST/OpenAI policy、provider credential置換、socket/peer identityの既存testが弱体化・skip・置換されていないことをsource/test auditで確認する。 | `focused-rerun` / affected package race testsはhermetic・deterministic・boundedであり、controlと既存経路の回帰を同じcandidateで検出できる。 |
| QA-006 | AC-6 | candidate diffが許可8 pathだけで、追加＋削除が概ね1,000行規模に収まり、Kakesu runtime、Go workspace、Schema、dependency、生成物、new socket/unit/listener、Git helper/launcher/environment injection、Git Smart HTTP/push/Approval/Tailscale/Passkeyを含まないことを確認する。READMEがcontrolのsecret非露出・対象外実環境を過大に保証しないことを確認する。DEVのroot `make check`/`git diff --check`、Reviewerのcandidate/root check証跡をcommand/cwd/resultまで独立監査し、QAはfull checkを重複実行しない。 | `evidence-review` / candidate diff、README、focused test本文とcandidate-bound DEV/Reviewer証跡でscope・非対象・行数を監査できる。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜005を次の一回だけ実行する。testはpeer-derived subject、allowlist/model gate、control/CONNECT分離、same Registry issue→consume→reuse reject、subject-bound revoke、secret non-leak、raceの少なくとも一つを壊すと失敗する必要がある。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/capabilitycontrol ./internal/connectsession ./internal/egressservice ./internal/capability
```

zero exitだけでは不十分であり、candidate不一致、対象test欠落、skip/弱体化、unbounded/non-deterministic fixture、required negative assertionの欠落は該当ケースをPASSにしない。root `make check`はDEV一回とReviewerの証跡監査に限り、QAはこのfocused Go suite以外を再実行しない。

## 境界・異常・回帰

- control subjectはpeer contextだけから導出し、Agent入力、socket address、PID/GID、request bodyから生成・補完しない。GitHub repository scopeはconfig allowlistと既存policyへ、OpenAIの個別model検査は既存policyへ委譲する。
- control requestはstrict・上限付き・Content-Length必須・一接続一操作でcloseする。chunked/upgrade/keep-alive/early bytes/複数操作はissuer、credential、network到達前に固定拒否する。
- issue、consume、revokeは同じRegistryのみを使い、capabilityのopaque handle以外のcredential/grant internalをAgentへ返さない。retry、rollback、cache、persistence、audit persistence、追加goroutine、new listener/socket/unitを追加しない。
- raw handle、credential、URL、config、provider detail、lower errorを診断またはresponseに含めない。existing CONNECT/TLS/HTTP、policy、credential resolver、socket/peer identityの意味を変更しない。
- 実GitHub/OpenAI、DNS/TLS、NSS/別UID、systemd socket、VPSはnot-runであり、Task外部状態を変更しない。hermetic PASSでそのlive事実を代替・主張しない。
- 許可path外、Kakesu runtime、Go workspace、Schema、dependency、生成物、Git helper/launcher/environment injection、Git Smart HTTP/push/Approval/Tailscale/Passkeyの変更はscope failureとして分類する。failureをDEV不具合と決めつけず、candidate、environment、requirement、証跡のいずれかに分類する。

## 実装後の再確認

- [x] 同一candidateでQA-001〜006を独立に評価し、指定focused Go race suiteを一回だけ実行した。
- [x] peer-derived subject、GitHub allowlist、OpenAI nonempty-model gate/既存policy委譲、control/CONNECT分離、same Registry lifecycle、subject-bound revoke、fixed non-leak diagnosticsのnegative failure-detectionを確認した。
- [x] candidateの許可8 path、約1,000行、対象外差分を監査し、DEV root `make check`/diff checkを確認した。並行していた独立ReviewerのPASSも後続で完了し、QAはfull checkを重複実行していない。
- [x] 実GitHub/OpenAI、DNS/TLS、NSS/別UID、systemd socket、VPSをlive-e2eとして計画・PASS主張していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。peer subject、bounded control、shared Registry lifecycle、revoke、non-leak、既存CONNECT回帰とlive境界を定義。 | `approved` |
