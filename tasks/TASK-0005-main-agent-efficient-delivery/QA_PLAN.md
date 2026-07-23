---
task_id: "TASK-0005"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-15"
revision: 2
implementation_reviewed_at: "2026-07-15T07:53:05+10:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0005 QA PLAN

## 方針

マージ済みproduct `main`にある新規skillを、構造、利用時の指示、参照資料、独立した利用者視点のforward test、repository品質gateで確認する。時間短縮の主張だけでPASSにせず、PLAN→DEV→REVIEW→QA、role分離、native spawn、親所有Git境界を緩和しないことを観測する。TASK-0003の集約計測値はraw traceの再計算ではなく、指定値・出典・限界を参照資料に正しく記録することを対象とする。

## 前提と環境

- QA対象はmain Agentが`--no-ff`で統合したproduct `main`の`.agents/skills/run-efficient-task-delivery/`である。work repositoryのTask証跡は読み取り専用の根拠として扱い、QAはskill、role定義、launcher、Git設定を変更しない。
- `python`、Git、product `make check`、`/Users/autotaker/.codex/skills/.system/skill-creator/scripts/quick_validate.py`が利用可能であること。QAはmainにマージされた差分、`REVIEW_RESULT.md`、`HANDOVER.md`を実装後に照合する。
- `quick_validate.py`はSKILL.md frontmatterのYAML、必須`name`/`description`、hyphen-case、name長、descriptionの型・長さ・angle bracketを検証する。`agents/openai.yaml`と本文・referenceの意味内容は別ケースで確認する。
- forward testはDEVとは異なるfresh Agentに、期待結論を渡さず「この低リスクTaskをgateを維持して進める手順」を依頼する。Agentには実装、Git操作、再委譲を許可せず、短い手順と根拠だけを返させる。
- `make check`はbuild、test、lintおよびtabletop viewer生成を含み、生成物・lock・依存取得への書込みを要し得る。QA sandboxで実行不能な場合も未実行をPASSに読み替えない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | SKILL.mdがTask開始・継続・遅延振り返りでtriggerし、main Agent向けの簡潔な命令形手順を定義する | SKILL.md frontmatterと本文を直接レビューし、`quick_validate.py .agents/skills/run-efficient-task-delivery`を実行する。 | frontmatterはname `run-efficient-task-delivery`と、3場面を含む明確なdescriptionだけを持つ。本文は命令形、500行未満で、必要時にreferenceを一段で読む簡潔な手順であり、validatorはexit 0となる。 | command、exit status、frontmatter/行数、reference link。 |
| QA-002 | 事前読取り、scope固定、完了契約、出力抑制、局所→完全検証、権限・依存関係の扱いを含む | SKILL.mdの各手順をTASK/PLAN/QA計画/適用AGENTS/Wiki、許可ファイル、role別完了条件、出力上限、局所検証、最終完全検証、既知の権限・依存失敗の見積り・分類へ対応付ける。 | 全項目を実行前に固定し、必要十分な要約を返すよう指示する。各gateで無目的な全文出力、60秒固定poll、既知失敗の反復、完全検証の重複を推奨しない。一回の最終`make check`を品質gateとして残す。 | reviewed heading/文言、対応表、差分照合。 |
| QA-003 | native spawn契約、gate、role分離、Git/lock境界を効率化で緩和しない | SKILL.mdをCodex Agent Model Routing、DECISION-0001/0004、適用AGENTS.mdと照合する。`task_name`、`agent_type`、異種roleの`fork_turns="none"`、observed model/effort不一致、fallback、child Git、親のlock/commit、main merge、FAIL分類の記述を検索する。 | `task_name`を追跡ID、`agent_type`をrole選択として区別する。異種roleでは`fork_turns="none"`を明示し、不一致/内部spawn不能/agent_type欠落時は停止してrequested/observed/runtimeを証跡化した後に限定fallbackをmainが判断する。不一致成果、CLI fallbackの通常化、gate省略、childのstage/commit/merge/.git書込み、親責務の委譲を勧めない。 | 検索語と該当箇所、根拠Wiki、差分照合。 |
| QA-004 | `agents/openai.yaml`が最小interface metadataで、default promptがskill名を明示する | YAMLをparseし、キー、quoted string、display name、short description、default promptを検査する。 | top-levelは`interface`のみで、`display_name`、25–64文字の`short_description`、`$run-efficient-task-delivery`を明示する短い`default_prompt`だけを持つ。icon、brand、dependency、policyなど未要求metadataはない。 | parsed YAML、field名/文字数、prompt、余分なkeyがない結果。 |
| QA-005 | retrospectiveが指定観測値、主要因、改善仮説、出典と限界を記録する | `references/2026-07-15-task-0003-retrospective.md`をレビューし、TASK-0005本文と照合する。 | 43分30秒、約2,152万token累積入力、97.6% cache、`wait_agent` 31回/22分16秒、tool出力約25万文字、主要因、43分から22–28分への改善仮説を記録する。sourceがTASK-0005の背景/受け入れ条件であること、raw trace不在、tokenが課金/実トークンと同一でないこと、因果未証明、目標は強制SLO/合否gateでないこと、将来比較手順を明記する。 | 指定値の抜粋、source/limit節、将来比較手順。 |
| QA-006 | 独立Agentがskill単体からgateを維持した効率化手順を導ける | freshな独立Agentにskill pathと短い現実的Task依頼だけを渡し、実装せず進め方を返すforward testを一回実施する。意図する結論、想定FAIL、既存PLANの診断は渡さない。 | 回答は適用規約と承認済み証跡を読んでscopeを固定し、PLAN→DEV→REVIEW→QAを順に進める。native `agents.spawn_agent`、role分離、異種roleの`fork_turns="none"`、model/effort観測、parent-owned Gitを維持する。gate省略、CLI fallbackの通常化、child Git、無制限の並列/再委譲を勧めない。 | exact prompt、fresh-thread/role、短い出力、各期待との判定。 |
| QA-007 | `git diff --check`がPASSする | merge済みmainで対象差分を確認して`git diff --check <merge-parent>..HEAD`（または同等の実装差分）を実行する。 | exit 0で、対象skill 3ファイル以外の意図しない変更、末尾空白、space-before-tab、conflict markerがない。 | comparison range、command、exit status、changed paths。 |
| QA-008 | repository `make check`がPASSする | merge済みmainで`make check`を一回実行する。 | exit 0。build、test、lint、tabletop viewer生成、最終`git diff --check`を含むrepository品質gateが成功する。 | command、exit status、簡潔なtarget summary、生成差分の有無。 |

## 境界・異常・回帰

- 旧skillやglobal Codex skill、`.codex` role、launcher、lock、hook、Task Schema、telemetry、work repository Wikiを変更していないことを差分で確認する。product変更はSKILL.md、agents/openai.yaml、retrospective referenceの3ファイルに限定される。
- malformed YAML、許可外frontmatter、non-hyphen skill name、欠落trigger、長大/非命令的本文、broken reference、metadataの余分なkey/短すぎる説明、指定計測値・限界の欠落、fallback通常化、role/Git境界の緩和、forward testでのgate省略をFAIL候補として扱う。
- `make check`や`quick_validate.py`のFAILは、実装と承認済みTask/PLANの照合後に`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`へ分類する。DEVへの自動帰責、差し戻し、revert、バグ化の判断はmain Agentに委ねる。

## 実施不能時の扱い

- merge済みmain、review証跡、fresh Agent、Python、依存済みtoolchain、lock、または生成先書込みが利用できない場合、影響ケース、再現command、阻害した依存、観測済み範囲をQA_RESULTに記録し、PASSとしない。
- read-onlyの構造/文書確認は継続する。`make check`がQA sandboxのlockまたは生成物書込みで阻害された場合、main Agentが適切なlock/権限環境で同一commandを再実行した結果を別証跡で確認する。これは環境所見であり、製品acceptance failureやDEV不具合とは自動分類しない。
- 実装後の差分が本計画の操作にだけ影響する場合は手順を更新できる。期待結果または試験範囲を変える必要がある場合は、理由を改訂履歴へ残しmain Agentの承認を得るまで、既存期待を置き換えない。

## 実装後の再確認

- [x] 実装差分、REVIEW_RESULT、HANDOVER、merge済みmainを確認した。
- [x] QA-001〜QA-008の操作を現行実装へ対応付けた。
- [x] 期待結果または試験範囲の変更有無を確認した。
- [ ] 期待結果または範囲を変更した場合、理由を改訂履歴へ記録しmain Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-15 | | 初版placeholder | `pending` |
| 2 | 2026-07-15 | qa-agent-terra-medium | TASK、承認済みPLAN、skill-creator規約、関連Wikiに基づく実装前QA計画 | `approved` |
| 2 | 2026-07-15 | qa-agent-terra-medium | 実装後のmain照合、REVIEW_RESULT・HANDOVER確認、QA-001〜QA-008実施。期待結果・試験範囲は変更なし。 | `approved` |
