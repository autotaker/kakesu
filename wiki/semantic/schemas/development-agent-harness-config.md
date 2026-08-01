---
kind: schema
title: Development Agent Harness Config Policy
---

# Development Agent Harness Config Policy

## 問い

Development Agent Harness の version 1 設定を、値を漏らさず read-only に事前検証し、危険な設定ファイルを処理対象から除外するにはどうするか。

## 設定形式と意味検証

version 1 の設定は strict JSON として扱う。unknown field、duplicate key、trailing data、未知または不正な version は受理しない。JSON decoder の field-disallow だけでは duplicate key を検出できないため、token scan と typed strict decode を分けて実施する。

意味検証は parser の成功と別に行う。network の既定は deny であり、path、user、network、allowlist の制約に反する値は拒否する。許可される `check-config` は設定値を出力せず、成功時には固定された検証結果だけを返す。通常の起動経路は fail-closed のままとする。

## 設定ファイルの FD 基準 policy

設定ファイルは 64 KiB を上限として、regular file だけを対象にする。open 時は no-follow と nonblocking を併用し、read 前後に同じ FD の属性を検査する。symlink、directory、FIFO、size や mode の不正な file は拒否する。

FIFO は read-only open であっても block し得るため、file type を確認する前の open にも nonblocking が必要である。read 前後の検査は attribute race を抑えるが、実行環境固有の race 耐性は独立した QA で確認する対象として残る。

## 生成・検証時の path 境界

Autoconf の既定値では設定例に `${prefix}` が残り、絶対 path の fixture にならない。設定例を install 後に検証する場合は、README で指定する absolute `--sysconfdir`、`--localstatedir`、`--runstatedir` を明示する。未展開変数を受理するために設定契約を緩和しない。

## 関連

- [TASK-0035 HANDOVER](../../../tasks/TASK-0035-dev-agent-config-policy/HANDOVER.md)
