---
kind: schema
title: Safety Contract Completion Preflight
---

# Safety Contract Completion Preflight

## v2 opt-in contract

安全契約をv2として検査するPLANは `safety_contract_version: 2` を明示し、リポジトリ相対file pathの配列で `safety_contract_planned_paths` と `safety_contract_generated_paths` を宣言する。空配列は各種別に変更がないことを表す。

pathは空文字、絶対path、`..` を含むpath、directory、globを受理しない。各配列内および両配列間の重複も受理しない。通常予定pathは安全契約の通常allowlistに、生成pathは生成専用allowlistにそれぞれ一致しなければならない。`docs/99-glossary-index.md` は生成pathとしてだけ許可される。

## 検査の時点と束縛

DEV開始前のpreflightはGit履歴を必要とせず、v2のversionと両path宣言をfail-closedで検査する。新しい安全契約の完了では、HANDOVERの`candidate_commit`だけを候補の正本とし、Main承認済みQA_PLAN、既存4項目がすべて`pass`の`safety_checks`、および`safety_checked_at`を要求する。製品用のREVIEW_RESULT/QA_RESULTのPASSは要求も生成もしない。

merge中は`MERGE_HEAD`を候補と照合する。完了後はmainから到達できる一意な厳密two-parent no-ff mergeを導出し、その第2親がHANDOVER候補であることを確認する。候補のmerge-baseからのname-status差分について、全pathが二つの宣言の和集合に含まれること、宣言された生成pathが全て差分に現れることを確認する。空差分、rename/copy、未宣言path、main-managed path、許可外pathを拒否する。

`merged_commit`、candidate/merge tree、digestは新しい完了経路で転記・要求・照合しない。

計画外の差分は後付けallowlistで通さない。PLANを再承認するか、別Taskへ分離する。

## 互換性境界

version fieldを持たない安全契約は従来の固定allowlistによる検査を維持する。v2 fieldをversionなしで混在させる状態と、未知versionはlegacyとして扱わず拒否する。

公開済みlegacy安全契約だけは、旧`HANDOVER.status: safety_contract_complete`と既存`task.merged_commit`の組合せを互換入力として受理する。新Taskが旧statusだけで候補正本を省略する偽装は拒否する。legacyに残るtree/digest等は遡及変更せず、互換入力として再検証しない。

## 関連

- [TASK-0031 handover](../../../tasks/TASK-0031-safety-completion-preflight/HANDOVER.md)
