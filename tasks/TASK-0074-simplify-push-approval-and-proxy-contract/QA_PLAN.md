---
task_id: "TASK-0074"
change_class: "safety_contract"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T09:05:23Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0074 QA PLAN

## 方針

TASK.md の Planning input packet を唯一の期待値根拠とする。これは製品実装を伴わない safety contract 変更であるため、candidate-bound の文書・backlog 差分と既存 safety contract 検査を独立監査する。実VPS E2E は次の製品Taskの受入であり、本Taskの PASS に代用しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-074-01 | AC-1, AC-2 | candidate の設計書差分を TASK の grant 境界と逐語比較する。exact repository の次の `git-receive-pack` 一回、agent instance/UID・workspace・短TTL・原子的一回消費・revoke を必須とし、ref/SHA/manifest/body 一致を認可根拠から外していることを確認する。同一repository内の内容差替え残余リスクの受容と、別repository・別主体/workspace・再使用・期限後・REST転用の拒否、repository 限定 GitHub App write 権限を確認する。 | `evidence-review` / 文書契約の肯定・禁止要件は、固定 candidate と TASK-first expectation の対照で判定でき、実装又は live E2E は対象外である。 |
| QA-074-02 | AC-3 | candidate の設計書差分で、通常の Git read/GitHub REST/OpenAI が最小の framing・hop-by-hop header・secret 境界を除く未解釈 stream 転送であり、push は repository と `git-receive-pack` のみ分類して本文を解析しないことを確認する。Unix peer identity、host allowlist、Opaque handle、credential 置換、TLS CONNECT/local CA、timeout/concurrency/header 上限、secret-free audit を維持し、`approvalmanifest`、old/new SHA・ref・force/delete・remote old SHA、Git wire/pkt-line本文照合、strict OpenAI JSON field/model/store/stream、GitHub `/repos/{owner}/{repo}` parser、upload-pack本文/response Content-Type、JSON response/2xx、全量buffer/1MiB、Policy→Transaction→Exchange→Forwarder の削除対象を漏れなく明記することを確認する。 | `evidence-review` / retained boundary と明示的 deletion inventory は candidate-bound 文書差分で完全に監査できる。 |
| QA-074-03 | AC-4, AC-5 | candidate diff と `backlog.yaml` を監査する。UI の主認可文言が「このrepositoryへの次のpush一回」で、branch/commit/ref/SHA は参考表示に限られること、TASK-0070 が superseded、TASK-0071〜0073 が履歴を保持した repository 単位への移行対象、次の単一製品Taskが薄いproxy＋承認後pushの実VPS vertical E2E を最優先とすることを確認する。既存 TASK-0070〜0073 証跡が未変更で、変更が許可された設計書・backlog・TASK-0074 計画証跡だけに限られ、製品コード/test/config/dependency/Schema/生成物、新規check/field/wrapper がないことを確認する。既存 `make test-process`、contract scope audit、`make lint-docs`、`make check`、`make task-check TASK=TASK-0074`、`git diff --check` の candidate-bound 結果を監査する。`make check` は既存 safety contract 検査としてのみ扱い、製品QA PASS の証拠にはしない。 | `evidence-review` / backlog の将来状態、公開証跡の不変性、許可path と既存 safety contract 検査は candidate diff と実行証跡から確認できる。 |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- ref/SHA/manifest、branch/commit表示、provider protocol の意味検査を認可根拠又は必須検査として残す候補は FAIL とする。
- 実VPS、proxy、grant、credential 又は既存 TASK-0070〜0073 公開証跡を変更する候補は scope failure とし、本Taskで live E2E PASS を作らない。
- 許可path 外、製品成果物・依存・Schema・生成物、新規機械check・証跡field・互換wrapper の追加、又は既存検査の失敗/未実施は FAIL 又は blocked とする。

## 実装後の再確認

- [ ] fixed candidate に対し QA-074-01〜003 の TASK-first diff/evidence audit を完了した。
- [ ] 既存 safety contract 検査、`git diff --check`、許可path と既存公開証跡の不変性を確認した。
- [ ] 実VPS E2E を本Taskの PASS に使わず、次の製品Taskへ残した。
- [ ] 期待結果又は範囲を変更した場合、main Agent の承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-first safety contract QA計画。 | `main-agent-sol-high / 2026-08-02T09:05:23Z` |
