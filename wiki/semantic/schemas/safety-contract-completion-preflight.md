---
kind: schema
title: Safety Contract Completion Preflight
---

# Safety Contract Completion Preflight

## v2 opt-in contract

安全契約をv2として検査するPLANは `safety_contract_version: 2` を明示し、リポジトリ相対file pathの配列で `safety_contract_planned_paths` と `safety_contract_generated_paths` を宣言する。空配列は各種別に変更がないことを表す。

pathは空文字、絶対path、`..` を含むpath、directory、globを受理しない。各配列内および両配列間の重複も受理しない。通常予定pathは安全契約の通常allowlistに、生成pathは生成専用allowlistにそれぞれ一致しなければならない。`docs/99-glossary-index.md` は生成pathとしてだけ許可される。

## 検査の時点と束縛

DEV開始前のpreflightはGit履歴を必要とせず、v2のversionと両path宣言をfail-closedで検査する。Done検査では、候補の実差分pathの全てが二つの宣言の和集合に含まれること、宣言された生成pathの全てが候補差分に現れることを確認する。既存のno-ff、tree/digest、rename/copy、空差分、非`AMD`の検査も維持する。

計画外の差分は後付けallowlistで通さない。PLANを再承認するか、別Taskへ分離する。

## 互換性境界

version fieldを持たない安全契約は従来の固定allowlistによる検査を維持する。v2 fieldをversionなしで混在させる状態と、未知versionはlegacyとして扱わず拒否する。

## 関連

- [TASK-0031 handover](../../../tasks/TASK-0031-safety-completion-preflight/HANDOVER.md)
