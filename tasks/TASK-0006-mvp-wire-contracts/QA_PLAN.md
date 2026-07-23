---
task_id: "TASK-0006"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-18T12:19:27+09:00"
revision: 1
implementation_reviewed_at: "2026-07-18T12:40:01+09:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0006 QA PLAN

## 方針

マージ済み`main`を対象に、canonical JSON Schemaを変更せず、同一の共有JSON fixture・同一のlocal設定をGo Core、Python Memory、Rust Governanceが同じ結果で検査することを確認する。単体テストのPASSだけで代替せず、fixtureのraw-byte digest結合、explicit `null`と欠落の区別、UTC `Z`表記、設定のlocal-only制約を利用者から観測できる契約として確認する。UDS bind/listener、ACK、Inbox/Outbox、payloadの意味Schema検証は本Taskの対象外であり、試験がそれらを要求する場合は期待値を拡張しない。

## 前提と環境

- 実施対象はTASK-0006を`main`へマージした後のクリーンなworktree。QA開始時に`git status --short`、対象commit、`REVIEW_RESULT.md`、実装差分を記録する。
- 共有fixtureは`fixtures/wire/envelope/`の5ファイル、catalogは`schemas/draft-v0`、正例の対象Schemaは`common/error.schema.json`とする。fixture/config/testの配置が実装で同等に変更された場合は、機能同値であることを差分で確認して実パスを証跡へ記録する。
- 正例は空でない`message_id`、異なる`from_component`/`to_component`、`correlation_id`、**明示した**`"causation_message_id": null`、末尾`Z`のUTC RFC3339 `occurred_at`、`schema_id`/revision/`sha256:` lowercase 64 hex digest、JSON object payloadを持つ。
- focused testは実装で追加されるGo/Python/Rustのfixture/config testを、各言語の標準runnerから実行する。全体回帰はリポジトリrootで`make check`を一度実行する。ネットワーク接続、UDS socket bind、秘密情報は不要である。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | 3言語が同じ有効fixtureを受理し、同じ4種の不正fixtureを拒否する | 下表の5 fixtureをGo focused test、`uv run --project memory pytest`のfocused test、`cd governance && cargo test --locked`のfocused testで読む。 | 全15判定が期待どおり。言語ごとの複製fixture、言語差のある期待値、skip/xfailはない。 | 各runnerのcommand、対象test名、5件×3言語のPASS出力。 |
| QA-002 | fixtureの必須metadata、UTC、object payload | `valid.json`を構造確認し、3言語でdecode/validateする。`causation_message_id`をfieldごと削除した一時コピーも各focused testへ入力する。 | 正例は受理される。明示`null`は値として受理され、欠落はnull補完されず全言語で拒否される。 | fixture抜粋、欠落変異のtest名と3言語のreject結果。 |
| QA-003 | 起動設定から各Plane endpointとcatalog rootを解決し、デフォルトがlocal-only | `configs/local/wire.json`を各設定readerに渡す。rootから解決したcatalog pathと3 endpointを検査する。 | `version: 1`、相対`schemas/draft-v0`、正確に`core-runtime`/`memory-service`/`governance-service`の3 component、非空・相対・相互非重複のendpointを得る。設定に秘密値、network host/port、socket bindはない。 | 設定fixture、各language config focused testのPASS、解決値のassertion。 |
| QA-004 | Schema変更時、fixtureのschema参照/digest未更新なら失敗 | test用一時catalogへ`common/error.schema.json`をコピーし、対象ファイルの**一バイトのみ**を変える。fixtureのschema ID/revision/digestは正例のままとし、各language readerで検査する。 | 3言語すべてdigest mismatchとして拒否する。JSON整形・canonicalization・改行正規化をして受理しない。一時catalogはtest終了時に除去される。 | 変異前後の`sha256sum`、変異内容/offset、各languageの失敗assertion。 |
| QA-005 | 必要な実装・テスト・文書、および独立レビューと全体検査 | 実装差分と`REVIEW_RESULT.md`を読み、schema artifactが未変更であること、shared fixture/config/testが追加されていることを確認後、rootで`make check`を実行する。 | reviewerがPASSで、build/test/lint/tabletop/document/process/diff checkを含む`make check`がPASS。Schemaを変更していないためschema revision更新・tabletop scenario変更は要求しない。 | commit SHA、review evidence、`make check`終了0と要点出力、`git diff --check`結果。 |

### 共有fixture matrix（QA-001）

| fixture | 変異する唯一の契約 | Go | Python | Rust |
|---|---|---:|---:|---:|
| `valid.json` | なし。explicit null、UTC `Z`、catalog ID/revision/raw-byte digest、object payloadを満たす。 | accept | accept | accept |
| `invalid-same-route.json` | `from_component == to_component` | reject | reject | reject |
| `invalid-non-utc-timestamp.json` | RFC3339としてparse可能だが末尾`Z`でないoffset（例`+09:00`） | reject | reject | reject |
| `invalid-schema-reference-digest.json` | catalogに一意に結合しないschema ID/revision/digest | reject | reject | reject |
| `invalid-non-object-payload.json` | `payload`がobject以外 | reject | reject | reject |

## 境界・異常・回帰

- `causation_message_id`は正例のexplicit `null`と、fieldを完全に欠落させた変異を別々に扱う。後者を各言語でrejectすることをQA-002で確認する。`null`を欠落に同一視する実装は`implementation_defect`である。
- timestampは`Z`を含むUTCのみを正例にする。`+00:00`、`+09:00`、timezoneなし、parse不能値の各変異について、実装testが少なくともoffset（`+09:00`）を明示的にrejectし、追加変異もfail closedであることをfocused testから確認する。UTC instantが同じでもwire表現が`Z`でなければrejectする。
- schema referenceは`$id`、`x-schema-revision`、対象ファイルのraw bytes SHA-256を結合する。未知ID、revision不一致、digest prefix欠落、uppercase hex、digestのみ不一致、catalog内の0件/複数件解決をrejectするtestがあることを確認する。payload内容をJSON Schemaで意味検証することは要求しない。
- configは正例に加え、version不正、component不足、component余剰/未知、absolute catalog root、absolute endpoint、`..`を含むendpoint、空endpoint、重複endpointを各readerがrejectすることを確認する。endpointが相対でも3件未満、4件以上、または相互重複なら受理しない。
- fixtureが全言語の同じリポジトリroot共有pathから読まれること、catalog mutationが元のSchema/fixtureを編集しない隔離temporary directoryで行われることを実装差分とtestで確認する。既存の言語固有constructor testも回帰しない。
- スコープ境界として、UDS listener/connect、socket作成、retry/ACK、durable Inbox/Outbox、秘密値、HTTP/TCPを新たに要求しない。これらの副作用がQA中に観測された場合は`implementation_defect`候補としてmain Agentへ報告する。

## 実施不能時の扱い

| 状況 | 記録する証跡 | 仮分類 | 標準差し戻し先 |
|---|---|---|---|
| shared fixture、catalog、config、focused testのいずれかが欠落、またはmatrix結果が言語間で異なる | command、fixture名、実際/期待、3言語の結果 | `implementation_defect` | DEV |
| QA手順が実装済みの承認PLANと矛盾、または要求されたfixture名/期待値がTask・PLANにない | Task/PLAN該当箇所、差分、矛盾理由。期待値は勝手に変更しない。 | `qa_plan_defect` または `requirement_gap` | QA または PLAN/main Agent |
| runner、lockfile、依存取得、権限、破損したtest環境により再現不能 | exact command、exit code、最初の失敗抜粋、環境情報、最小再現。少なくとも別言語/静的確認で切り分ける。 | `environment_issue` | QA または基盤Task |
| TASK-0006以外の既存testが、変更前はPASSした基線からFAILへ遷移 | 基線/対象commit、失敗test、最小再現、差分関連性 | `regression` | DEV（必要ならrevertをmain Agentが判断） |

分類は暫定であり、最終判断はmain Agentが`QA_RESULT.md`または`HANDOVER.md`に理由とともに残す。主要受け入れ条件、build、または安全なlocal-only境界を満たせない場合は、原因をDEVの責任と決めつけず上表の証跡で判定し、revertまたはバグTask化をmain Agentへ判断依頼する。

## 実装後の再確認

- [x] マージ済み`main`の実装差分、commit、`REVIEW_RESULT.md`を確認した。
- [x] shared fixture 5件、explicit null/欠落、UTC `Z`/offset、catalog ID/revision/raw-byte digest、one-byte temporary catalog mutationを実装に即した操作で確認した。
- [x] 3 component、root-relative catalog、相対・非重複endpointとconfig負例を3言語で確認した。
- [x] Go/Python/Rust focused testを実行し、matrix全15判定を記録した。
- [x] `make check`を一度実行し、結果を記録した。
- [x] 期待結果または試験範囲の変更有無を確認した。変更する場合は理由を改訂履歴に記録し、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-18 | qa-agent-terra-medium | 実装前QA計画。3言語5 fixture matrix、null/UTC/catalog raw-byte digest/config境界、実施不能時分類を具体化。 | `main-agent-sol-high` |
