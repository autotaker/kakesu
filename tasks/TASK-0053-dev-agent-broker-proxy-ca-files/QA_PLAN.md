---
task_id: "TASK-0053"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T14:39:31Z"
revision: 3
implementation_reviewed_at: "2026-08-01T15:16:53Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0053 QA PLAN

## QA scope

期待値正本は TASK.md の `Planning input packet` だけとする。PLAN、実装案、DEV の自己申告から期待値を導かない。同一 candidate の許可 path
`tools/dev-agent-harness/internal/brokercredentials/`、`tools/dev-agent-harness/internal/providercredentials/providercredentials_test.go`、
`tools/dev-agent-harness/README.md` だけを独立確認する。providercredentials の既存fixtureへCA二fileを生成する直接回帰追随以外、`proxyca` の意味、listener/session/broker composition、設定/CLI/service binary、生成物、依存を変更していないことを確認する。

実 provision/generate/rotate/trust/client/VPS は authority、実 secret、OS identity、外部作用及び安全な cleanup が未定義であるため live-e2e を blocked とする。hermetic fixture、公開 CA copy 又は leaf issue の PASS はこれらを置換しない。

## Cases

| Case ID | 対象AC | 確認内容と failure detection | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | 固定 basename が既存4件へ `proxy-ca-cert.pem` と `proxy-ca-key.pem` を重複なく固定順で加えた6件だけであることを source/test と candidate diff で監査する。欠落、空入力、不正 path、nil/zero/corrupt state が panic せず固定 `ErrLoad` となり、全6件成功時だけ Bundle を返すこと、既存 credential/JWT/Format の公開契約を変えず余分な入力を解釈しないことを failure-detection fixture で確認する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result、candidate diff |
| QA-002 | AC-2 | CA二fileを既存と同じ一 directory fd から一度だけ `openat`/no-follow で読むことを監査する。absolute clean directory、effective UID non-root、directory owner/mode/type、file owner/mode/regular/size、read 前後の dev/inode/mode/uid/gid/size/nlink/mtime/ctime 一致を確認し、symlink/FIFO/device、hardlink、group/other access、metadata/content race、別 path reopen/fallback を実際に失敗検出する test を確認する。 | focused-rerun | bundled package race test と candidate source/test |
| QA-003 | AC-3 | 既存 credential 検証完了後に CA certificate/key を `proxyca.New` へ一回だけ渡し、single PEM、self-signed CA、ECDSA P-256、key 一致、現在有効及び leaf 発行余命を proxyca に再実装せず委譲することを監査する。package-private clock seam を `proxyca.ClockFunc` が呼出し時にも動的参照すること、malformed/multiple/mismatched/non-P256/non-CA/expired/leaf余命不足が固定 `ErrLoad` となり partial Bundle を返さず、path/PEM/key/certificate/time/parser/OS detail を error/Format に出さず raw input byte slice を Bundle が保持しないことを failure-detection fixture と source で確認する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result、candidate diff |
| QA-004 | AC-4 | `ProxyCAAuthority()` だけが成功 Bundle の検証済み Authority を返し、nil/zero/corrupt Bundle は nil を返すことを監査する。PEM/private-key/signer/file-path/raw-input accessor、marshal 又は秘密 detail を Format/error に足していないことを確認する。返された Authority が公開 CA certificate の独立 copy と exact 2 hosts の fresh leaf だけを issue できること、caller の公開 CA slice 変更が後続結果を変えないことを failure-detection fixture で確認する。 | focused-rerun | bundled package race test と candidate source/test |
| QA-005 | AC-5 | valid six-file fixture で ClientID、InstallationID、OpenAI key、JWT と新 Authority を同時利用でき、CA failure では既存値だけの Bundle を返さないことを監査する。並行した公開 CA copy/leaf/JWT 利用が state/secret を混線させず、既存 credential/JWT security negative tests を削除・緩和していないことを確認する。 | focused-rerun | bundled package race test と candidate source/test |
| QA-006 | AC-6 | hermetic race test が six-file all-or-nothing、Linux file policy/TOCTOU、CA parse/key/validity negative、nil/Format/non-leak accessor、two-host issue/public copy、既存 credential/JWT regression、parallel isolation を実際に失敗検出できることを監査する。candidate-bound root/harness check、distcheck、README lint、candidate launcher root `make check` は DEV 証跡だけを監査し、QA は再実行しない。差分が許可6 paths内で299 additions/9 deletions（追加＋削除1,000行以下）であり、providercredentials変更がCA二file fixture追加だけ、testを弱体化していないことも確認する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result、candidate diff |
| QA-007 | 対象外 | actual provision、CA generate/rotate/renew/reload/watch、public CA 配置/OS trust、TLS client、実 GitHub/OpenAI、service restart、実 broker/agent、VPS を確認する。 | live-e2e — blocked | 必要な authority、実 secret/UID、external environment と安全な cleanup が未提供。hermetic PASS で代替しない。 |

## Execution rule

QA-002、QA-004、QA-005 は同じ一回の focused-rerun 観測に束ねる。QA は `tools/dev-agent-harness` を cwd として、candidate に対し次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/brokercredentials
```

QA は root/harness `make check`、`make distcheck`、lint、追加 test 又は rerun を実行しない。QA-001、QA-003、QA-006 は candidate source/test、HANDOVER、DEV command/result の独立 evidence audit だけを行う。package race test が指定した negative/boundary の failure を検出できない、candidate/tree と証跡が一致しない、又は fixture が hermetic/deterministic/bounded でない場合、該当 focused-rerun は blocked 又は FAIL であり evidence-review PASS に置換しない。

## Result criteria

各 case を planning input packet と candidate-bound evidence に照らして記録する。focused-rerun では command、cwd/cache、exit status、実行 test、coverage と failure-detection evidence を残す。source/evidence audit は fixed six basenames、一 directory fd/openat/no-follow、nlink==1 と read 前後 metadata race、all-or-nothing、`proxyca.New` 一回委譲、ClockFunc の動的 clock、raw secret 非保持/非公開、nil/Format、public CA copy、two-host issue、既存 credential/JWT regression、parallel isolation を明示的に確認する。

失敗を実装不具合と決めつけず、`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`、又は evidence 不足として根拠付きで分類する。QA-007 は安全に実行可能となるまで blocked のままとする。

## 実装後の再確認

- [x] candidate source/test、HANDOVER、DEV check evidence を独立確認した。
- [x] 指定 package race test を candidate で一回だけ実行した。
- [x] fixed basenames、descriptor/no-follow、metadata/hardlink/race、all-or-nothing、proxyca delegation/dynamic clock、non-leak API、public-copy/two-host、credential/JWT/parallel isolation の failure detection を確認した。
- [x] 変更が許可 path と追加＋削除1,000行以内に収まり、既存 dependency/package の意味を変えていないことを確認した。
- [x] provision/generate/rotate/trust/client/VPS live-e2e を PASS に置換せず、期待値又は scope を変更していないことを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | QA (Terra/medium) | Planning input packet に基づく独立 QA 計画 | `approved` |
| 2 | 2026-08-01 | QA (Terra/medium) | 6-file必須化で直接壊れる既存providercredentials fixtureのCA二file追加を許可pathへ明記し、candidate実測6 paths/299 additions/9 deletionsへ監査範囲を補正。期待値・製品scopeは不変。 | `approved` |
| 3 | 2026-08-01 | QA (Terra/medium) | PLAN frontmatter correctionによるcandidate commit rebaseへ再固定。旧candidateとの`tools/`差分は空で、期待値・製品bytesは不変。 | `approved` |
