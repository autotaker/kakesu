---
task_id: "TASK-0026"
change_class: safety_contract
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 2
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0026 QA PLAN

## 方針

これは製品実装ではなく必須開発統制を変更する安全契約の計画である。TASK本文だけを根拠に、完成差分を独立に静的照合する。製品`candidate_commit`/`candidate_tree`、製品テスト、`QA_RESULT.md`のPASSは対象外とする。

## 前提と環境

- 対象は製品リポジトリ内の規範文書、Task template、関連skillだけであり、製品成果物・宣言済み製品依存は変更しない。
- 照合対象はTASKが列挙するAGENTS、Agent責務、review/QA文書、Task template、関連skillと、それらに残る子AgentのGit操作に関する全面禁止である。
- 本計画の結果は安全契約の計画レビュー証跡であり、製品QAのPASSではない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | `qa_execution_mode` / 理由 | 操作 | 期待結果 | 必要証跡 |
|---|---|---|---|---|---|
| SC-001 | Reviewer/QAが自ら軽微と判断した指摘をTask worktreeで修正・stage・commitでき、Task branch取り込み確認でPASSにできる。DEV差戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。 | `evidence-review` / 規範文書の静的変更だけで受け入れ真実を確認できる。 | 対象差分とTASKを突合し、Reviewer/QAに限定した軽微修正・Task branch取り込み後のPASSを明記し、上記の追加ゲートを必須とする記述がないことを検索する。 | 裁量は閉じた許可リスト・SLOC上限・追加チェックリストへ置換されず、軽微修正の解消経路が一貫する。 | 対象パス、検索語、検索結果、差分、実行環境、exit、差分ダイジェスト。欠落または矛盾はPASSにしない。 |
| SC-002 | 挙動、要件、安全境界を変えると担当Agentが判断したとき通常差戻しへ戻す。 | `evidence-review` / 判断分岐は文書の静的整合で確認できる。 | 正常経路と通常差戻しの境界を差分横断で照合する。 | 軽微修正の例外が挙動・要件・安全境界の変更へ拡張されない。 | 対象パス、該当文言、差分、検索結果、exit、差分ダイジェスト。影響不明はPASSにしない。 |
| SC-003 | Mainだけがmainへのmergeを所有し、子Agentはmainへ直接merge/pushしない。 | `evidence-review` / 権限境界は規範文書の静的照合で確認できる。 | Mainのmain統合所有と子Agentのmain直接操作禁止を全対象規範で照合する。 | Reviewer/QAのTask branch commit許可はmain統合・push権限を与えない。 | 対象パス、検索語、検索結果、差分、exit、差分ダイジェスト。権限境界の曖昧さはPASSにしない。 |
| SC-004 | 既存の全面的な子Agentのstage/commit/merge/`.git`書込み禁止との矛盾が全規範から消える。 | `evidence-review` / 受け入れ真実は全規範の静的探索である。 | `rg`で製品リポジトリの規範（AGENTS、`docs/development/`、`templates/task/`、`.agents/skills/`、`.codex/`）を検索し、子Agent一般を対象にstage/commit/merge/`.git`書込みを全面禁止する残存文言を確認する。必要なら同義語でも再検索する。 | Reviewer/QAの軽微修正に矛盾する全面禁止は残らず、DEV等へ不要にGit権限を拡張しない。 | 検索対象・語句・一致箇所の判定、差分、exit、差分ダイジェスト。未探索範囲、残存矛盾、対象外への権限拡張はPASSにしない。 |
| SC-005 | DEV案、独立REVIEW/QA、修正後の判断、merge tree確認を証跡化できる。 | `evidence-review` / Git履歴とTask証跡の静的照合で受け入れ真実を決定できる。 | Task branchの最終candidate、Reviewerによる直接修正commit、Mainのno-ff merge、candidate/merge tree、PLANとTASK-first QA_PLANの独立作成記録を照合する。 | 各ロールの判断が同一Taskへ追跡でき、candidate treeとmerge treeが一致する。製品QA PASSは要求しない。 | candidate/merge commit・tree、親関係、planning review、Main承認、対象検査digest。欠落・tree不一致・担当混同はPASSにしない。 |

## 境界・異常・回帰

- 全面禁止を消す変更が、Reviewer/QAの軽微修正以外の子Agentへstage/commit権限を与える場合、またはmainへの直接merge/pushを許す場合は`requirement_gap`としてPLANへ戻す。
- 「軽微」の閉じた列挙、SLOC上限、追加承認チェックリストを導入した場合はTASKの対象外としてFAIL候補にする。
- 製品コード、テスト、runtime/build設定、Schema、依存、生成物が差分に含まれる場合は安全契約経路を停止し、製品変更へ再分類する。

## 実施不能時の扱い

- 規範の探索範囲、検索結果、または変更意図を確定できない場合はblockedとし、製品QA PASSや別モードのPASSで代替しない。
- FAIL候補はimplementation_defect、qa_plan_defect、requirement_gap、environment_issue、regressionを根拠付きで分類し、最終判断はMainが記録する。DEV責任とは推定しない。

## 実装後の再確認

- [ ] TASK本文と完成差分を独立に突合した。
- [ ] SC-001からSC-005の静的証跡を記録した。
- [ ] 製品成果物・依存の変更がないことを確認した。
- [ ] `QA_RESULT.md`に製品PASSを作成しないことを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | QA Agent | TASK-first 初版 | `main-agent approved` |
| 2 | 2026-07-20 | Main Agent | TASK末尾の証跡化条件へSC-005を明示対応。期待結果は不変更。 | `main-agent approved` |
