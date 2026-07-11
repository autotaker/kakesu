---
kind: case-pattern
title: Privilege Laundering via Child Task
---

# Privilege Laundering via Child Task

作業上の親が、Spawnした子のExternal Effectを承認できると、親自身の意図を子の要求として表現して自己承認する経路が生まれる。([権限モデルV2レビューEpisode](episode://design/authority-v2-review))

## 発生条件

- 親Agentが子Taskを生成できる。
- 子TaskがExternal Effectを要求できる。
- 子の要求が作業上の親へ承認Escalationされる。
- Effectのoriginではなく直近requesterだけを見る。

## 典型的な進行

1. 親が直接実行できないEffectを望む。
2. 親がその処理を子Taskへ委譲する。
3. 子がEffectを要求する。
4. 要求が親へ上がる。
5. 親が「子の要求」として承認する。

## 起こりうる帰結

- Spawnによって実質権限が増える。
- 同一目的系統内で自己承認が成立する。
- 監査上のoriginが隠れる。

## 対策

- 作業階層とGovernance Planeを分離する。
- Policy Judgeを作業Agentから独立させる。
- `origin_task_id`とdelegation chainを保存する。
- AuthorityをPolicyから解決する。
- CredentialをEffect Gatewayだけが保持する。

## 適用限界

Sandbox内の実装方針やTask Contractの変更を親が判断すること自体はこのPatternに該当しない。問題はSandbox外へのEffect許可を同じ親子系統で行う場合である。

## 関連

- [Task Ownership](../schemas/task-ownership.md)
