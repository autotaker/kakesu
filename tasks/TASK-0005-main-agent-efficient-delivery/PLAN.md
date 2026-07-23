---
task_id: "TASK-0005"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-15"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規リポジトリ内スキルの手順文書、UI metadata、過去セッションの参照資料だけを追加する局所的で機械検証可能な変更であり、runtime、role TOML、launcher、lock、hook、Git権限、Task Schemaを変更しないため。"
approved_dev_profile_risk_signals: []
planned_implementation_files: 1
planned_implementation_lines: 5
estimate_points: 1
---

# TASK-0005 PLAN

## DEV profile選定証跡

```yaml
profile: luna-xhigh
model: gpt-5.6-luna
reasoning_effort: xhigh
decision_rule: "局所的な文書／metadata変更で、契約・Schema・security・concurrency・migration・runtime変更がなく、機械検証可能ならluna-xhigh"
reason: "追加するのはmain Agent向けSKILL.md、最小interface metadata、TASK-0003の計測参照資料のみで、既存の実行機構と責務境界は変更しない"
```

DEV中に、既存のrole契約との矛盾を解くために`.codex`、launcher、lock、hook、Task Schema、または運用repositoryのプロセスを変更する必要が判明した場合、DEVは変更せず停止する。main Agentはrisk signalを記録し、`sol-high`へのpromotionとPLAN再承認の要否を判断する。降格はしない。

## 受け入れ条件の具体化

| 条件 | 完了時の観測方法 | 根拠 |
|---|---|---|
| トリガーと簡潔な手順 | `SKILL.md` frontmatterがTask開始・継続・遅延振り返りを明示し、本文が命令形かつ500行未満であることを確認する。 | TASK受け入れ条件、`skill-creator` |
| 事前読取りとスコープ固定 | スキルが適用される`AGENTS.md`、TASK、承認済みPLAN、承認済みQA計画、必要なWiki／Decisionを先に読み、許可ファイルと完了条件を固定するよう指示する。 | Task Delivery、Work Repository Boundary |
| native spawn契約を維持 | スキルが`task_name`を追跡ID、`agent_type`をrole選択として区別し、異種roleで`fork_turns="none"`、observed model/effort不一致時の停止・証跡化を要求する。 | Codex Agent Model Routing、DECISION-0004 |
| gate・責務境界を維持 | PLAN→DEV→REVIEW→QA、DEVとReviewer/QAの分離、子のGit禁止、親のlock/scope/hook/stage/commit/post-check、mainだけの承認・merge・FAIL分類を緩和しない記述を確認する。 | Task Delivery、DECISION-0001、DECISION-0004 |
| 効率化の実行内容 | 一体ずつの内部Spawn、role別の完了契約、必要十分な出力、局所検証後に一度の完全検証、既知の権限・依存失敗の事前見積りと分類を含むことを確認する。 | TASK-0005の背景・設計観点 |
| 回顧資料 | 参照資料が指定された計測値、主要因、43分から22–28分への非強制の改善仮説、出典、raw trace非保持とtoken計数の限界を分離して記録する。 | TASK-0005の背景・受け入れ条件 |
| UI metadata | `agents/openai.yaml`が`interface`の`display_name`、25–64文字の`short_description`、`$run-efficient-task-delivery`を明示する短い`default_prompt`だけを持つ。 | `skill-creator/references/openai_yaml.md` |
| 機械検証とforward test | `quick_validate.py`、`git diff --check`、`make check`がPASSし、freshな独立Agentがスキル単体からgateを残す手順を導ける。 | TASK受け入れ条件、`skill-creator` |

## 関連Wikiと判断

- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md) と [DECISION-0001](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md) に従い、効率化は順次gate、独立Reviewer、main merge、merge後QAを省略しない。QAのFAILはmain Agentが分類する。
- [Codex Agent Model Routing](../../wiki/semantic/schemas/codex-agent-model-routing.md) と [DECISION-0004](../../wiki/decisions/DECISION-0004-multiagentv2-role-startup.md) に従い、内部`agents.spawn_agent`を標準経路とする。`task_name`はrole selectorではなく、異種roleには`agent_type`と`fork_turns="none"`を使う。不一致の子成果は採用せず、限定fallbackを通常化しない。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md) と [DECISION-0002](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md) に従い、製品側スキルはproduct repository、Task証跡とWikiはwork repositoryに置く。スキルはTask証跡、Wiki本文、work側設定を更新しない。
- `DECISION-0003`は`DECISION-0004`に置換済みの履歴であるため、現行起動契約の根拠には使わない。TASK-0003 HANDOVERは現行契約とsandbox未観測の運用注意を補足する。

## 設計

### 選択案

`.agents/skills/run-efficient-task-delivery/`を、main AgentがTaskを開始、継続、または遅延を振り返る際の短いワークフロー・スキルとして新設する。`SKILL.md`には繰り返し使う判断手順だけを置く。

1. 開始時に適用規約、Task、承認済みPLAN／QA計画、必要なWikiを読み、role、責務、許可ファイル、検証、完了契約を一枚の実行境界として確定する。
2. 子を一体ずつ内部spawnし、正規`agent_type`、異種roleの`fork_turns="none"`、observed model/effortを確認する。未利用・不一致・spawn不能時だけ停止・証跡化後に既存の限定fallback可否をmainが判断する。
3. 各roleへの依頼は、所有ファイル、禁止操作、完了時に返す短い証跡、局所検証、次gateへ渡す条件を明記する。全文出力、同じ状態の60秒固定poll、既知の権限／依存失敗の無目的な再試行、各段階での完全検査を避ける。
4. 変更者が局所検証を行い、ReviewerとQAはそれぞれ独立した観点で必要な再確認を行う。`make check`は計画済みの最終品質gateとして一回実行し、失敗はQAガイドラインで分類する。
5. 遅延後だけ参照資料の指標を使い、wait、tool出力、完全検証、後発lint、権限待ちを分けて振り返る。目標時間は改善仮説であり、合否gateやAgentの評価指標にはしない。

TASK-0003固有の数値は、SKILL.mdを肥大化させず`references/2026-07-15-task-0003-retrospective.md`へ分離する。参照資料には、TASK-0005の背景／受け入れ条件が記録する観測値をsourceとして明記し、raw session traceがこのrepositoryにないこと、累積tokenは課金や実トークン消費と同一視できないことを明記する。

### 代替案と不採用理由

- 計測値を`SKILL.md`へ直接置く案は、毎Taskで不要な長い文脈を読み込ませるため不採用。段階的開示として参照資料に分離する。
- gate、独立role、mainのmerge／FAIL分類を削って時間短縮する案は、TaskとDECISION-0001に反するため不採用。
- `make work-agent`またはCLI Agentを通常の効率化入口にする案は、DECISION-0004の内部spawn標準経路に反するため不採用。fallback条件は既存契約の引用に限定する。
- `.codex` role、launcher、hook、lock、telemetry収集を変更する案は、対象外であり低リスク文書／metadata Taskを越えるため不採用。
- 完全検証を常に省略する案は、品質gateを失うため不採用。局所検証を先行させ、計画済みの完全検証を重複なく実行する。

### 責務と境界

- main Agentはスキルの利用、PLAN承認、role起動の観測照合、fallback判断、lock／commit／merge、QA FAIL分類を所有する。
- DEVは承認済み`luna-xhigh` PLANの3ファイルだけを実装し、stage、commit、merge、`.git`書込みを行わない。
- ReviewerはDEVから独立して差分と検証を確認し、freshなforward testを実施する。QAはDEVから独立して承認済みQA計画に従い、マージ済みmainを受け入れ確認する。
- `skill-creator`は構成、frontmatter、metadata、validationの作成規約を提供する。`tabletop-debug-scenarios`は本TaskがPlane／Schema／scenario変更を含まないため適用しない。ただし同スキルの「機械検証と独立レビューを完了する」原則は既存開発規約と整合する。

### 不変条件

- 変更可能なproductファイルは、`.agents/skills/run-efficient-task-delivery/SKILL.md`、`agents/openai.yaml`、`references/2026-07-15-task-0003-retrospective.md`の3ファイルだけである。
- `SKILL.md` frontmatterは`name`と`description`だけを持ち、nameは`run-efficient-task-delivery`とする。本文は命令形で、triggerをfrontmatter descriptionへ含める。
- `agents/openai.yaml`には必要最小限のinterface metadataだけを置き、追加icon、brand、dependency、policyは追加しない。
- スキルはnative spawn、role分離、停止・証跡化、子Git禁止、親所有のlock／commit、mainだけのmergeを弱めず、CLI起動を提案しない。
- リポジトリの原データ不在を補うtelemetry、外部サービス、集計スクリプト、受け入れSLOは追加しない。

### 失敗時・移行・互換性

- 初期化、metadata生成、または`quick_validate.py`が失敗した場合、DEVは原因を3ファイル内の構造に限定して修正し、失敗結果をHANDOVER候補として返す。新規ツール、設定、検証回避策を追加しない。
- `make check`またはforward testが失敗した場合、Reviewer／QAはQAガイドラインに従い`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`を区別し、DEV不具合と自動帰責しない。
- 既存スキルは変更しない。新スキルは新規の明示／暗黙trigger候補であり、グローバルCodexスキルやユーザー設定の互換性を変更しない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `.agents/skills/run-efficient-task-delivery/SKILL.md` | documentation / skill instruction | - | main Agent向けの短い命令形ワークフロー、native spawn契約、完了契約、出力抑制、検証順序、権限見積り、遅延振り返りへの参照を追加する。 |
| `.agents/skills/run-efficient-task-delivery/agents/openai.yaml` | configuration | 5 | 必要最小限の`interface` metadataを生成し、`default_prompt`で`$run-efficient-task-delivery`を明示する。 |
| `.agents/skills/run-efficient-task-delivery/references/2026-07-15-task-0003-retrospective.md` | documentation / reference | - | TASK-0003の観測値、出典と限界、主要因、非強制の改善仮説、今後の比較手順を記録する。 |

見積もり対象はconfiguration 1ファイル・5行だけである。文書と参照資料は規則どおり対象外とする。

## 見積もり

`file_score = ceil(1 / 3) = 1`、`line_score = ceil(5 / 200) = 1`であるため、`estimate_points = 1`とする。実装コード、Schema、runtime設定、テスト、生成スクリプトは追加しない。

## 実装手順

1. DEVは製品worktreeで適用される`AGENTS.md`、TASK、承認済みPLAN／QA計画、`skill-creator`、role契約、TASK-0003 HANDOVERを再読し、3ファイル以外に変更が不要であることを確認する。
2. `skill-creator`の`init_skill.py`で`run-efficient-task-delivery`を`.agents/skills`配下に`references/`付きで初期化し、deterministic generatorへ`display_name`、`short_description`、`default_prompt`を渡す。placeholderを残さず、生成物を3ファイルに限定する。
3. `SKILL.md`を、開始・継続・遅延振り返りをfrontmatter triggerに含めた簡潔な命令形へ置換する。詳細な数値はreferenceへの一段だけのリンクにし、既存gateと起動契約を要約して変更しない。
4. referenceへ、43分30秒、約2,152万token、97.6%、`wait_agent` 31回／22分16秒、tool出力約25万文字を近似観測として記録する。sourceをTASK-0005の背景／受け入れ条件とし、raw trace不在、キャッシュ入力を含む累積値、因果未証明、22–28分が強制SLOでない限界を明記する。
5. `quick_validate.py`でskill folderを検証し、metadataがSKILL.mdと一致すること、referenceのリンクが有効であることを確認する。差分をscope表と照合し、`git diff --check`、`make check`を実行する。
6. Reviewerは別fresh threadの内部spawnで、skill本文とreferenceだけを与えずスキルpathと短い現実的Task依頼だけを与えるforward testを行う。依頼は「この低リスクTaskをgateを維持して進める手順を示す」とし、意図した回答・診断・結論を漏らさない。返答がgate省略、CLI通常化、role／Git境界の緩和を勧めないことを確認し、通常の独立レビューと`make check`を完了する。

## 検証計画

- `python /Users/autotaker/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/run-efficient-task-delivery`：frontmatter、必須name／description、hyphen-case、YAMLを確認する。
- metadata確認：`agents/openai.yaml`がquoted stringの最小`interface`だけで、25–64文字の`short_description`と`$run-efficient-task-delivery`を含む`default_prompt`を持つことを確認する。
- 文書レビュー：trigger、事前読取り、scope固定、一体ずつのnative spawn、role別完了契約、出力上限、局所→完全検証、既知権限／依存の分類、遅延振り返りがあることを確認する。`task_name`／`agent_type`、`fork_turns="none"`、mismatch停止、限定fallback、gate、role分離、Git／lock境界が現行契約と一致することを確認する。
- reference確認：指定した全計測値と、source、raw trace／token計数／因果の限界、主要因、22–28分を非強制の仮説とする説明、将来比較の手順が存在することを確認する。
- `git diff --check`：空白エラーがないことを確認する。
- `make check`：リポジトリ完全検証を一回実行する。失敗は出力と再現条件を保存し、QAガイドラインに従って分類する。
- 独立forward test：freshなReviewer Agentが、前提結論なしの実際の依頼からスキルを利用し、PLAN→DEV→REVIEW→QA、native spawn、role分離、親所有commit、限定fallbackを維持した手順を返すことを確認する。出力は成功／失敗と根拠だけを記録し、実装・Git操作・再委譲を依頼しない。

## 未解決事項

- なし。TASK-0003のraw session traceは現repositoryにないため、referenceではTASK-0005が提示する集約観測を唯一のsourceとして扱い、再現可能な原計測や課金値とは主張しない。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 設計観点と代替案を検討している。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV profile `luna-xhigh`、選定理由、risk signalなしを承認した。
- [x] DEV開始を承認した。
