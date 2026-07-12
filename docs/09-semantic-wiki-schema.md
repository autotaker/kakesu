# Semantic Wikiスキーマ設計

## 1. 方針

Semantic層はMarkdown本文を正とし、YAML frontmatterは文書種別と表示名だけにする。

```yaml
---
kind: schema
title: Task Ownership
---
```

IDはファイルパスから導出する。

```text
semantic/schemas/task-ownership.md
→ semantic://schema/task-ownership
```

更新日時はGit履歴、関連はMarkdown link、根拠は本文中のEpisode linkで表す。frontmatterへ`sources`、`status`、`maturity`、`related`を重複保持しない。

## 2. 四種類

| kind | 答える問い | 認知的役割 |
|---|---|---|
| `concept` | それは何か | 対象の意味、特徴、境界、典型例 |
| `schema` | 何がどう関係するか | 登場要素、役割、関係、制約の枠組み |
| `script` | 通常どう進むか | 典型的な時間展開、分岐、終了、失敗 |
| `case-pattern` | どんな条件で何が起きやすいか | Episode群から得た経験的パターン |

## 3. Concept

Conceptは辞書的一文ではなく、対象を識別・区別するためのまとまりである。

推奨見出しを示す。

```markdown
# Task

Taskは……。([episode://task/T-100](episode://task/T-100))

## 特徴
## Taskではないもの
## 境界事例
## 関連
## 代表例
```

Conceptには次を含める。

- 中核的特徴
- 近い概念との差
- 境界事例
- 他Concept / Schemaへのlink
- 代表Episode

実例: [../examples/semantic/concepts/task.md](../examples/semantic/concepts/task.md)

## 4. Schema

Schemaは複数Conceptを、1つの状況を理解する構造へ配置する。

推奨見出しを示す。

```markdown
# Task Ownership

## 登場主体
## 関係
## 構造上の制約
## 例外・未解決
## 関連するScriptとCase Pattern
```

Schemaは静的構造を主に扱う。手順の列挙だけならScriptへ分ける。

実例: [../examples/semantic/schemas/task-ownership.md](../examples/semantic/schemas/task-ownership.md)

## 5. Script

Scriptは典型的な出来事の進行順序を表す。

推奨見出しを示す。

```markdown
# Task Completion

## Trigger
## 標準進行
## 分岐
## 終了条件
## 典型的な失敗
## Variations
```

Scriptは絶対的なWorkflow定義とは限らない。実際のTaskがどのように進むと期待されるかを表す認知モデルである。

実例: [../examples/semantic/scripts/task-completion.md](../examples/semantic/scripts/task-completion.md)

## 6. Case Pattern

Case Patternは個別Episodeに近いが、複数状況で再利用できる経験的形を持つ。

推奨見出しを示す。

```markdown
# Privilege Laundering via Child Task

## 発生条件
## 典型的な進行
## 起こりうる帰結
## 対策
## 反例・適用限界
## 観測Episode
```

普遍的ルールのように断定しない。条件、反例、適用限界を本文に残す。

実例: [../examples/semantic/case-patterns/privilege-laundering.md](../examples/semantic/case-patterns/privilege-laundering.md)

## 7. 根拠リンク

frontmatterの`sources`は持たない。本文の説明単位ごとにEpisodeへlinkする。

```markdown
一つのOwnerは同時に一つの非終端Taskだけを処理する。
この制約はwaiting中の文脈競合を防ぐために採用された。
([episode://design/task-owner-exclusivity](episode://design/task-owner-exclusivity))
```

複数文が同じ根拠を共有する場合、段落末尾にまとめる。ページ末尾のSources一覧は補助であり、本文linkを正とする。

## 8. Link種別

```text
[Task](../concepts/task.md)
[Task Completion](../scripts/task-completion.md)
[episode T-100](episode://task/T-100)
[artifact](artifact://T-100/test-result)
```

Wiki AgentはMarkdown link graphを検索・探索に利用する。

## 9. ファイル配置

```text
semantic/
├── concepts/
│   └── task.md
├── schemas/
│   └── task-ownership.md
├── scripts/
│   └── task-completion.md
└── case-patterns/
    └── privilege-laundering.md
```

一ページに複数kindを混在させない。たとえばTask Ownershipページに完了手順を長く書かず、Task Completion Scriptへlinkする。

## 10. Semantic形成

新Episodeごとに新ページを作らない。

```text
New Episode
  → 既存Concept / Schema / Script / Patternで説明可能か
      ├─ Yes: 例・反例・適用範囲を更新
      └─ No : 既存モデルを調節、またはPattern候補を作成
```

### 同化

既存構造のまま、典型例・反例として追加する。

### 調節

既存構造では説明できないため、概念境界、役割、関係、Script分岐を修正する。

## 11. 成熟度の扱い

`maturity: stable`のような機械的ラベルはfrontmatterへ置かない。

不確実な理解は本文で表す。

```markdown
## 現在の理解

ReviewerはOwner Agent Runから分離した一時API sessionであるべきだが、同一モデル利用時に十分な独立性が得られるかは未確定である。

## 未解決

- Reviewerへ見せるWorkspace範囲
- 小TaskでのReview省略条件
```

これにより、何が不確実かを圧縮せず残せる。

## 12. ページ分割基準

別ページにする目安を示す。

- 1つの中心的問いに答えられる
- 他ページから独立してlinkされる
- 典型的な見出し構造が一kindに収まる
- 一Taskの具体的経緯ではなく複数Taskで再利用できる

一時的な議論はEpisodeか設計Artifactに残し、早期にSemanticページへ昇格させない。

## 13. Wiki Agent向け編集規約

- 現在TaskのObjectiveを書き込まない
- 一EpisodeのOwner assertionを一般則にしない
- 反例を削除しない
- 記述の根拠を段落単位でlinkする
- 同じ説明をfrontmatterと本文に二重化しない
- 既存linkを保つかredirectページを作る
- 歴史的経緯は必要な場合だけ短く説明し、詳細はEpisodeへlinkする

## 14. Harnessへ返すView

Wiki Agentはページ全文をそのままWork Agentへ渡さない。Taskに必要な抜粋を次の区画で生成する。

```text
Relevant concepts
Applicable schemas
Expected scripts
Case warnings
Relevant episodes
Open / contested points
```

過去ページの文章が現在の命令に見えないよう、Memory blockとして明示する。
