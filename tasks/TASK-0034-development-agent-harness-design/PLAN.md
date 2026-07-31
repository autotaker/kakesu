---
task_id: "TASK-0034"
change_class: safety_contract
status: approved
safety_contract_version: 2
safety_contract_planned_paths:
  - "docs/development/development-agent-harness.md"
  - "docs/glossary.yml"
safety_contract_generated_paths: []
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Credential isolation, network mediation, approval authorization, replay/TOCTOU, and recovery span multiple trust boundaries. The later document-authoring task must use the high-risk DEV profile even though this Task produces no product artifact."
approved_dev_profile_risk_signals:
  - "credential and authentication boundary"
  - "network egress and TLS mediation"
  - "asynchronous approval, one-use grant, replay and TOCTOU handling"
  - "external-service assumptions that require later live-VPS validation"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T08:45:10+10:00"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T08:45:00+10:00"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-01T08:45:20+10:00"
classification_approval_reason: "製品artifactを除外し、設計文書と用語sourceだけを予定差分とし、内容不変の生成索引を偽装しないsafety_contract v2 reconciliationであるため"
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# TASK-0034 PLAN

## 分類と差分契約

- `safety_contract` v2 とする。成果は外部開発基盤の設計文書だけであり、製品コード、テスト、runtime/build 設定、Schema、製品依存、生成製品入力/成果物、製品挙動は変更しない。
- 候補差分は設計文書と用語sourceの2ファイルに限定する。`docs/glossary.yml`は通常予定pathであり、用語generatorのindex出力はbase比で内容不変でなければならないため生成pathに宣言しない。rename/copy、Kakesu Plane/Schema/runtime/dependency、tabletop の変更は許可しない。
- 依存する外部仕様は packet の REF-1〜REF-7 を設計根拠として引用する。Kakesu 本体への採用判断は `pending` のままにし、ready になっても本Taskへ範囲追加せず、製品変更Taskを新規に再審査する。

## 変更設計と実施順序

1. 文書の冒頭に「Development Agent Harness」を Kakesu 本体外の開発専用提案として定義し、配置先、責任範囲、非採用境界、将来に製品へ採用する際の再審査トリガーを置く。構成要素と依存先をここから参照できるようにする。
2. 脅威モデルと信頼境界を先に固定する。owner/login、agent OS identity、Codex 実行環境、Credential Broker、Egress Proxy、Approval Service、Tailscale tailnet、GitHub/OpenAI を分離し、filesystem/environment/process/socket/network の各経路で Agent が実credentialへ到達できないことを明記する。
3. credential-mediated egress をプロトコル別に設計する。`gh`、Agent code の OpenAI API、Git Smart HTTP read 操作は opaque capability のみを入力とし、Broker/Proxy が許可済み host/repository/operation と expiry を検証して短命 credential に置換する。Git は custom credential helper の入出力と HTTPS transport の境界を示し、Codex 実credentialの例外を Agent code 用 capability と別枠にする。
4. push を非同期な永続状態機械として記述する。request 作成、policy evaluation、通知、human decision、one-use grant 発行、Agent 明示再実行、consume/audit、取消・期限切れ・復旧を状態と遷移で示す。grant は repo/ref/expected-old/new SHA/force/delete/policy version/expiry に束縛し、pending 時に Agent process を保持しない。
5. approval ingress を localhost backend + Tailscale Serve の tailnet HTTPS に限定して設計する。Funnel 無効、Tailscale identity/Grants、backend allowlist、毎回の Passkey user verification、通知の非権限性を別々に示す。
6. 横断的な fail-closed 規則を整理する。未知の capability、宛先・repo・operation 不一致、proxy/broker/approval 到達不能、identity/header/Passkey 不備、grant の stale/expired/replay、TOCTOU、監査不整合は拒否する。秘密はログ・監査・エラーに書かず、失効、端末紛失、回復時の隔離・再認証・再発行を定める。
7. 最小構成、段階導入、未決実装選択、検証matrixを末尾にまとめる。静的・fixture で確認できる設計検査と、VPS、tailnet、GitHub App、Passkey、実 Codex surface を必要とする live 検証を分離し、後者を文書PASSの代替にしない。
8. 公式参照一覧を設計判断にリンクする。各 REF の適用箇所、依存する事実、将来実装時に再検証する項目を対応付け、更新日と packet の固定参照を混同しない。
9. 設計文書で新規・更新した用語を `docs/glossary.yml` のsourceへ反映し、`uv run --project memory python scripts/validate-terminology.py --write` を実行する。同じgeneratorを再実行して差分が収束すること、および `docs/99-glossary-index.md` がbase比で内容不変であることを確認する。許可済み2パス以外、indexの変更、または未収束の差分は完了扱いにしない。

## AC-ID 対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | 外部開発基盤という名称・境界・採用再審査を文書の最初に固定する。 | `docs/development/development-agent-harness.md` | 1 | 本体との責任境界が一意でなければ文書を完成扱いにせず、製品採用の記述を削除または別Taskへ戻す。 |
| AC-2 | identity、secret、IPC、network を別々の到達不能性として定義する。 | 同上 | 2 | いずれかの実credentialが Agent に到達可能なら fail-closed とし、例外を追加せず境界設計を再検討する。 |
| AC-3 | opaque capability を唯一の Agent 入力にし、Broker/Proxy 側で限定・置換する。 | 同上 | 3 | protocol の検証点または短命化が特定できなければ、その egress 対応を未採用として記録する。 |
| AC-4 | push の束縛済み one-use grant と無保持の非同期遷移を定義する。 | 同上 | 4 | binding、consume 原子性、再実行のいずれかが未解決なら push を許可しない段階へ戻し、実装Taskのblocking decision にする。 |
| AC-5 | localhost-only approval backend と Serve/Grants/Passkey の多層確認を定義する。 | 同上 | 5 | Funnel、通知承認、identity-only 承認へ弱化する案は拒否し、実装選択が未定なら live 検証のblocked条件にする。 |
| AC-6 | 拒否既定、秘匿監査、失効・紛失・回復を横断規則にする。 | 同上 | 6 | fail-open、secret logging、replay/TOCTOU の未処理があれば設計レビューFAILとして PLAN へ戻す。 |
| AC-7 | 段階導入と検証matrixを実装選択・live VPS 要件から分離する。 | 同上 | 7 | 実環境依存事項を静的検査のPASSで代替している場合は、matrix を修正し live 項目を未実施/blocked と明記する。 |
| AC-8 | REF-1〜REF-7 を各判断へ追跡可能に結ぶ。 | `docs/development/development-agent-harness.md`、`docs/glossary.yml` | 8〜9 | 根拠のない設計主張または参照と判断の対応欠落は、主張を削除するか固定参照に戻って再確認する。用語generatorが未収束またはindex差分を出した場合は、差分を確定せずscope reconciliationへ戻す。 |

## 検証と引き渡し

- 文書作成前に `make task-preflight TASK=TASK-0034` を実行し、v2 path 宣言、TASK-first QA_PLAN、承認前提の不足を fail-closed で確認する。
- 文書作成後は、用語generatorを `--write` で実行して `docs/glossary.yml` を更新し、同じgeneratorの再実行で決定性と収束を確認する。`docs/99-glossary-index.md` はbase比で内容不変であることを検査し、許可済み2パス以外、index差分、または未収束差分があればscope reconciliationへ戻す。
- 設計境界の独立計画レビュー、対象の process/contract/docs 検査、`git diff --check`、`make check` を安全契約の証跡として実行する。製品DEV、製品REVIEW/QA PASS、tabletop 変更、実VPS操作は行わない。
- 失敗は `requirement_gap`、`qa_plan_defect`、`environment_issue`、`regression`、または設計文書の `implementation_defect` に分類し、差し戻し先の最終判断は Main が行う。

## 見積り

- 予定実装ファイル: 0、予定実装行: 0。文書だけであり見積り計算から除外されるため、`raw_points = 1`、`estimate_points = 1`。
- 後続実装の言語、process 分割、Codex credential 注入方式、Tailscale app capability header の利用可否は本見積りに含めない。これらは設計本文で未決として残し、実装Taskで再審査する。
