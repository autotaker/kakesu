---
task_id: "TASK-0001"
status: complete
completed_at: "2026-07-14T08:22:10+10:00"
---

# TASK-0001 HANDOVER

## 成果

- 製品リポジトリにTask単位のPLAN / DEV / QAプロセス、Agent責務、レビュー、QA、言語別ガイドラインを追加した。
- 独立workリポジトリにバックログ、6証跡、Semantic Wiki、Decision、Schema、カンバン／ロードマップを構築した。

## 主要な変更

- Task生成、規則ベース見積もり、フェーズゲート、worktree作成・解放、全Agent writer、Wiki Agent ingest、viewer生成をNode.js CLIとして実装した。
- 共通ロック、共有pre-commit、Ajv Schema、Decision不変性、commit親関係、HANDOVER digestで境界を機械検査する。
- Codex CLIを0.144.3へ更新し、Wiki Agentは既定で低コストな`gpt-5.6-terra`を使う。profileとmodelはランチャー単位で上書きできる。

## 検証結果

- 製品`make check`: pass
- work`make work-check`: pass
- 開発プロセス単体テスト: 7件pass
- 独立レビュー: P0=0、P1=0、pass
- Terra対応差分の独立再レビュー: P0=0、P1=0、pass
- viewer: 1280pxで7状態とEpic進捗を目視確認済み

## Decision

- workリポジトリは独立Gitの`main`一本とし、Wiki本文はWiki Agent、Schemaと権限境界はmain Agentが所有する。
- QA FAILはDEV責任と仮定せず、QA、DEV、PLANへの差し戻し、revert、バグ化をmain Agentが判断する。
- 見積もりはテストと文書を除外し、16ファイル、1,500行から8ポイントへrevision 2で再承認した。

## 既知の制約と未解決事項

- hook、work-agent、worktree rollback、merge親、Ajv負例のsubprocess統合テストは今後の改善候補である。
- 用語集インベントリの大規模な機械差分は、今後は独立Taskで扱う。

## 運用上の注意

- workリポジトリ初期化後は`make work-init`で共有pre-commitを有効化する。
- 証跡書き込みAgentは`make work-agent`または専用ランチャーから起動し、直接並行編集しない。

## Wikiへ引き渡す知識

### 再利用可能な知識

- main一本の運用リポジトリでは、Agentの編集開始からcommit後検証まで共通ロックを保持する。
- 実装前QA計画は実装後に再確認し、期待値変更には割当済みmain Agentの承認を要する。
- HANDOVER取り込み完了は、HANDOVER自体を書き換えずdigest付きreceiptの存在で判定する。

### 反例・失敗・注意点

- pre-commitはworking treeではなく完全stageを要求し、Wiki Agentのaction別許可パスとDecision履歴を保護する。
- blockedでも`resume_status`の復帰先フェーズ不変条件を維持する。

### 更新候補ページ

- `wiki/semantic/scripts/task-delivery.md`
- `wiki/semantic/scripts/wiki-ingestion.md`
- `wiki/semantic/schemas/work-repository-boundary.md`

## Bootstrap例外

- プロセス導入前のTaskであるため製品`main`上で実装し、DEVとQAを`main-agent-bootstrap`が兼任した。
- 独立`bootstrap-reviewer`による複数回レビューと全検査を代替ゲートとし、P0/P1をすべて解消した。
