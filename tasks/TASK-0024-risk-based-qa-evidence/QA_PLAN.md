---
task_id: "TASK-0024"
change_class: safety_contract
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 3
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0024 QA PLAN

## TASK-first baseline

この計画は、既存のPLAN、実装差分、既存QA_PLANから期待値を導かず、TASK.mdだけから先に固定した。QAの独立性は、DEVの結果を無批判に再利用せず、期待値、case選択、証跡完全性、testの失敗検出能力をQAが独立に判断することにある。自動testの再実行回数を減らすこと自体はPASS根拠ではない。

対象は開発統制を変更する安全契約であり、製品DEV、製品REVIEW_RESULT、製品QA_RESULTのPASSを作らない。TASK-first QA_PLAN、独立計画レビュー、Mainの承認、既存のDEV/Reviewer/QA分離、Main所有Git、FAIL分類は不変である。

## 方針とmode選択規則

各将来QA caseはDEV開始前に次の一つを理由と共に割り当てる。`evidence-review`はcandidate-bound DEV証跡の独立監査であり、testを実行しないことを許すmodeではない。証跡が不足、candidate/treeが一致しない、testが削除・弱体化されている、またはリスクが不明なときはPASSにせず、より強いmodeまたはFAILへfail-closedする。

| mode | 使用できる条件 | QAの最低操作 | fail-closed条件 |
|---|---|---|---|
| `evidence-review` | 低リスクかつ、実行済みDEV証跡だけでcaseの期待値を判断できる | case ID、candidate commit/tree、command/test、環境/fixture、cache条件、exit、artifact digest、未実施理由を突合し、testのnegative検出能力と弱体化の有無を差分で監査する | 証跡欠落・digest不一致・candidate不一致・期待値不明・高リスク信号 |
| `focused-rerun` | 高リスクでもhermetic、deterministicかつboundedに再現できるcase。例: IPC/Schema parser、race/concurrency、lifecycle state machine、persistence/error injection、fail-closed negative path | 独立したhermetic環境で対象command/testと明示したnegative/境界caseを再実行し、candidate/tree・環境・結果を記録する | hermetic性、決定性、boundednessのいずれかを示せない、実環境境界を必要とする、又は影響が不明 |
| `live-e2e` | real OS privilege/auth（sudo/PAMを含む）、install/deploy/generated config placement、external side effect/service、real restart/rollback/cleanup、environment-specific integration | 隔離・承認済みの実環境で前提、操作、観測、cleanup/rollbackを実行する | 前提が不明、環境を用意できない、又は安全なcleanupを保証できない場合はblocked。未実施をPASS扱いにしない |

高リスク信号は、認証認可、秘密、sudo/PAM、IPC、Schema、設定、依存、並行性、lifecycle、persistence、error/fail-closed、install/deploy/config生成、実権限、外部作用、rollback、または影響不明である。これらは`evidence-review`単独を禁止する。hermetic・deterministic・boundedの三条件を全て示せる高リスクcaseだけを`focused-rerun`とし、実OS/実環境/外部作用を必要とするcaseは`live-e2e`とする。影響不明又は安全なcleanupを保証できないcaseは`live-e2e`としてblockedにし、PASSにしない。

## 前提と環境

- QA AgentはTASK.mdだけで初版を作り、PLAN承認後・DEV開始前にMainが範囲と期待値を承認する。
- このTaskの実装対象は製品ではなく、`AGENTS.md`、`docs/development/{README.md,development-process.md,qa.md,agent-roles.md,code-review.md,task-management.md,git-worktree.md}`、`templates/task/{TASK.md,QA_PLAN.md,QA_RESULT.md,REVIEW_RESULT.md,HANDOVER.md}`、`run-efficient-task-delivery`、`run-lap30`、必要ならその検査/生成器である。Lap30 event Schemaは変更しない。
- 運用上の証跡スコープは`TASK.md`（契約と完成条件）、`PLAN.md`（candidate及び実装計画）、`QA_PLAN.md`（case/mode/期待値）、`backlog.yaml`（状態・assignee・candidate/merge関連の索引）である。これら、ならびにDEV/REVIEW/QA/HANDOVER証跡の役割を混同させず、既存Taskを遡及変換しない。
- 静的確認は製品リポジトリのroot、運用証跡確認は`WORK_ROOT=/home/ubuntu/git/agent-harness-work`で行う。変更された文書・template・skill・operational evidenceの用語と必須項目を同一candidate差分で確認する。
- `docs/99-glossary-index.md`は`docs/glossary.yml`を更新して決定的生成器を実行した結果だけを許容する。製品側`.agents/skills/run-efficient-task-delivery/agents/openai.yaml`と運用側`../agent-harness-work/.agents/skills/run-lap30/agents/openai.yaml`は、それぞれ対応skill metadataが実際に古くなった場合以外は変更しない。
- forward-testは将来の製品Taskを実際に作成・実装せず、文書、template、skill、validatorが次節のシナリオを矛盾なく要求・許可・禁止することを検査する。実環境E2Eが必要な将来caseを、この安全契約Taskの文書確認で実行済みとは扱わない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | mode | 操作 | 期待結果・必要証跡 |
|---|---|---|---|---|
| QA-001 | 各QA caseへの3 modeの事前割当と理由、曖昧/高リスクのfail-closed | evidence-review | QA_PLAN template、QA guide、process、skillsを検索し、3 modeの列挙、case単位理由、high-risk/unknown時の昇格を照合する。 | 全て同じ綴りと意味を使用し、3 mode以外・理由なし・曖昧な`evidence-review`を許さない。対象箇所と検索結果を証跡に残す。 |
| QA-002 | candidate-bound DEV証跡とQAの独立監査 | evidence-review | template/process/roles/skillを、case ID、candidate commit/tree、command/test、環境/fixture、cache無効化条件、exit、artifact digest、未実施理由、およびQAのcommit binding・完全性・negative/弱体化監査に照合する。 | DEV自己承認だけでPASSにできず、各必須項目の欠落・digest/commit不一致・test弱体化をFAIL又は再実行へ導く。 |
| QA-003 | REVIEWとQAの同一candidateからの独立・並行評価 | focused-rerun | `README.md`、development process、roles、code review、task management、git-worktree、AGENTS、template、skillを追跡し、REVIEW/QAが同一candidate commit/treeを明記し、相互PASSを前提にしない並行開始条件を確認する。既存の独立性とMain所有Gitも検索する。 | 並行は同一candidateへの独立評価だけに限られ、DEVとの役割分離、Mainの承認・Git所有、FAIL分類を緩めない。candidateが異なる結果の組合せは禁止される。旧来の「REVIEW後にQAだけを開始」「QAはマージ済みmainだけ」「candidate QA前にマージ」を必須化する規範文は残らない。 |
| QA-004 | REVIEW修正後の`qa_carry_forward`の限定条件とMain記録 | focused-rerun | carry-forwardのtemplate/手順を読み、許可条件と禁止条件を表にして、forward negative scenarioを突合する。 | Mainだけが旧新commit、tree/diff、影響case、再実行証拠、理由を記録した場合だけ移送できる。QA FAIL、acceptance/QA_PLAN変更、高リスク信号、test削除/弱体化、影響不明では必ずfocused/full再QAとなる。 |
| QA-005 | merge tree同一時の引継ぎと環境依存だけのマージ後確認 | focused-rerun | merge tree/candidate tree同一性の判定、環境依存caseの識別、マージ後確認の条件を文書とtemplateで追跡する。 | tree同一かつ環境依存caseなしだけが全面マージ後QA省略の候補である。install/deploy/config生成、実権限、外部作用、rollbackはマージ後確認を維持する。tree差異又は不明は省略不可。 |
| QA-006 | template/skill/processの整合と既存Task証跡形式の保全 | evidence-review | 全対象パスを横断検索し、mode、証跡項目、carry-forward条件、candidate/merge tree、FAIL分類、`TASK.md`/`PLAN.md`/`QA_PLAN.md`/`backlog.yaml`の役割を対応表で比較する。`TASK.md` templateの完成条件、`QA_RESULT.md`の用途、Task完了条件を特に確認する。既存Task template frontmatterと既存Task証跡の可読性も確認する。 | 用語又は必須項目の矛盾がなく、既存case/証跡を遡及的に無効化しない。`QA_RESULT`を「マージ後だけ」、Task完了を「無条件のマージ後QAだけ」、又はcandidate QA前のマージとして固定する文言を残さない。FAIL分類は`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`のまま。 |

## forward-test scenarios（正常・境界・異常・互換性）

| シナリオ | 種別 | 操作 | 期待結果 |
|---|---|---|---|
| FT-001: 低リスクのcandidate-bound unit case | 正常 | 完全なDEV証跡を持つ低リスクcaseに`evidence-review`を割り当てる想定で、QAがcase/commit/tree/command/environment/cache/exit/digestとnegative testを監査する手順を追う。 | QAは再実行なしでも独立監査の根拠を記録できるが、DEVの自己申告だけでPASSしない。 |
| FT-002: 証跡欠落又はdigest/commit不一致 | 異常 | command、cache条件、artifact digest、candidate treeのいずれかを欠落・不一致にした想定を当てる。 | `evidence-review` PASSを拒否し、欠落を分類して`focused-rerun`、`live-e2e`、又は適切なFAILへ進める。 |
| FT-003: 高リスクcaseの二分岐と曖昧case | 境界/negative | (A) hermetic・deterministic・boundedなIPC/Schema parser、race/concurrency、lifecycle state machine、persistence/error injection、fail-closed negative path、(B) real OS privilege/auth・sudo/PAM、install/deploy/generated config placement、external side effect/service、real restart/rollback/cleanup、environment-specific integration、(C) 影響不明又は安全なcleanupを保証できないcaseをそれぞれ`evidence-review`へ割り当てようとする。 | 全分岐で`evidence-review`を拒否する。Aは`focused-rerun`、Bは`live-e2e`、Cは`live-e2e`としてblockedとなり、未実施又はunsafeなままPASSにしない。 |
| FT-004: REVIEW/QA並行とcandidate混線 | 異常 | 同一candidateで相互PASSなしに開始する経路と、REVIEWがcandidate A、QAがcandidate Bを評価する経路を比較する。 | 前者は許可、後者は結果を組み合わせられず再評価又は明示的なcarry-forward判断が必要。 |
| FT-005: 低リスクREVIEW修正 | 正常/境界 | 文言・非挙動の明示的低リスク修正後、Mainが旧新commit、diff、影響case、再実行証拠、理由を記録する想定を追う。 | 全許可条件を満たす場合に限り`qa_carry_forward`を記録できる。QAの削除・暗黙継承にはならない。 |
| FT-006: carry-forward禁止修正 | negative | QA FAIL対象、acceptance/QA_PLAN変更、auth、secret、sudo/PAM、IPC/Schema/config/dependency、concurrency/lifecycle/persistence/error/fail-closed、test削除/弱体化、影響不明を一つずつ含める。 | いずれもcarry-forward不可。影響caseの限定再実行又は全再QAを要求する。 |
| FT-007: merge後環境依存 | 回帰/実環境境界 | candidate tree=merge treeであっても、install/deploy/config生成、実権限、外部作用、rollbackを含むcaseを想定する。 | 全面QA省略条件を満たさず、該当環境依存caseのマージ後確認を維持する。 |
| FT-008: 既存証跡との互換 | 互換性 | 過去Taskの既存QA_PLAN/QA_RESULTを更新しない前提で、templateの追加項目と新手順が既存形式を必須の遡及変換なしに読めるか確認する。 | 既存証跡を破壊せず、新規Taskだけが拡張された記録を使用する。Lap30 event Schemaも変更しない。 |
| FT-009: 規範文書・Task完成条件の否定検査 | negative/互換性 | `README.md`、process、roles、code-review、task-management、git-worktree、AGENTS、TASK/QA/REVIEW templates、skillを読み、(a) REVIEW→QAの順次強制、(b) QAをマージ済みmain限定、(c) QA_RESULTをマージ後限定、(d) Task完了を無条件のマージ後QA限定、(e) candidate QA前のマージ、のいずれかが残るか確認する。 | 5つ全ての旧規範が除去又は新規契約に置換される。一方で同一candidate binding、独立性、Main所有Git、high-risk/環境依存の再実行、FAIL分類は残る。 |
| FT-010: glossary/skill metadataの生成境界 | negative | 文書語彙の変更後に`docs/glossary.yml`を検査し、必要なら決定的生成器で`docs/99-glossary-index.md`を生成する。製品側`run-efficient-task-delivery/agents/openai.yaml`と運用側`run-lap30/agents/openai.yaml`の各変更時は、対応skillの表示名、短い説明、default promptとのmetadata stale根拠を個別に確認する。 | indexの手編集は許可されず、generatorの再実行でdriftがない。対応metadataがstaleでないいずれかの`openai.yaml`差分はscope逸脱であり、片方のstaleをもう片方の変更理由に流用しない。 |

## exact static checks

実装後の独立レビューでは、candidate差分を対象に次を実行し、command、対象commit/tree、exit、要約を残す。

```sh
git diff --check <approved-base>...<candidate>
rg -n 'evidence-review|focused-rerun|live-e2e|qa_carry_forward|implementation_defect|qa_plan_defect|requirement_gap|environment_issue|regression|candidate (commit|tree)|merge tree' \
  AGENTS.md docs/development templates/task .agents/skills scripts Makefile
if rg -n '順次.*(REVIEW|レビュー).*QA|QA.*(マージ済み|merged).*main|QA_RESULT.*マージ後|マージ後QAが完了|candidate QA.*前.*マージ' \
  AGENTS.md docs/development/{README.md,development-process.md,qa.md,agent-roles.md,code-review.md,task-management.md,git-worktree.md} \
  templates/task .agents/skills/run-efficient-task-delivery; then exit 1; fi
if git diff --name-only <approved-base>...<candidate> -- docs/99-glossary-index.md | grep -qx 'docs/99-glossary-index.md'; then
  git diff --name-only <approved-base>...<candidate> -- docs/glossary.yml | grep -qx 'docs/glossary.yml'
fi
if git diff --name-only <approved-base>...<candidate> -- .agents/skills/run-efficient-task-delivery/agents/openai.yaml | grep -qx '.agents/skills/run-efficient-task-delivery/agents/openai.yaml'; then
  rg -n 'display_name|short_description|default_prompt' .agents/skills/run-efficient-task-delivery/{SKILL.md,agents/openai.yaml}
fi
if git -C ../agent-harness-work diff --name-only <approved-work-base>...<work-candidate> -- .agents/skills/run-lap30/agents/openai.yaml | grep -qx '.agents/skills/run-lap30/agents/openai.yaml'; then
  rg -n 'display_name|short_description|default_prompt' ../agent-harness-work/.agents/skills/run-lap30/{SKILL.md,agents/openai.yaml}
fi
uv run --project memory python scripts/validate-terminology.py
make check
make task-check TASK=TASK-0024
make work-check WORK_ROOT=/home/ubuntu/git/agent-harness-work
```

negative `rg`はヒットが0件でなければFAILとする。文脈上同じ旧規範を別表現で残していないかも、列挙した全規範パスを人手で確認する。`docs/99-glossary-index.md`が差分にある場合は、`docs/glossary.yml`も同一candidateで変更され、`python scripts/validate-terminology.py --write`をcleanなcandidateで再実行した後にindex driftが0件であることを確認する。製品側`run-efficient-task-delivery`又は運用側`run-lap30`の`openai.yaml`が差分にある場合は、対応するskill metadataが実際にstaleだったことをファイルごとに記録する。

加えて、差分に製品コード、製品test、runtime/build設定、製品Schema、製品依存、Lap30 event Schemaが含まれないことを確認する。`make check`又は`make work-check`の既存失敗は、変更との因果、再現性、ログを確認して`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`のいずれかに分類し、DEV起因と決めつけない。

## 実施不能時の扱い

- static checkを実行できない場合は、実行しなかったcommand、阻害要因、影響case、再現条件を記録し、PASS根拠に置き換えない。環境・基盤が原因なら`environment_issue`候補としてMainへ渡す。
- live environmentの承認、隔離、secret、安全なcleanup/rollbackがない場合は`live-e2e`を実行しない。未実施理由と残余リスクを記録してblockedとし、当該caseは決してPASSにしない。MainがTask全体の未完了リスクを別途受理する判断と、caseのPASS判定を混同しない。
- TASK、PLAN、実装差分、template、skillの期待値が矛盾する場合は、都合よく手順を変更せず`requirement_gap`又は`qa_plan_defect`としてPLAN/QAへ戻す。

## 実装後の再確認

- [ ] 同一candidate commit/treeを基準に、実装差分、DEV証跡、REVIEW結果を確認した。
- [ ] 各caseのmodeと理由を再確認し、高リスク又は不明なcaseをfail-closedで昇格した。
- [ ] forward-test scenariosとexact static checksを実行又は実施不能として記録した。
- [ ] 操作手順を現行実装に合わせた。
- [ ] 期待結果または試験範囲の変更有無を確認した。
- [ ] 期待結果または範囲を変更した場合、改訂履歴へ理由を記録し、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | QA Agent | TASK-first初版。mode選択、candidate-bound証跡監査、carry-forwardの禁止条件、merge後境界、forward scenarios、静的検査を固定。 | `main-agent approved` |
| 2 | 2026-07-20 | QA Agent | expanded normative scope、TASK/PLAN/QA_PLAN/backlogのoperational evidence、旧順次/merge-after規範のnegative checks、glossary生成・skill metadata境界を追加。 | `main-agent approved` |
| 3 | 2026-07-20 | QA Agent | focused-rerun/live-e2eのhermetic/実環境discriminatorを明確化し、FT-003を二分岐化、run-lap30のstale-only metadata検査を追加。 | `main-agent approved` |
