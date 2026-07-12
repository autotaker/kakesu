---
kind: case-pattern
title: Privilege Laundering via Child Task
---

# Privilege Laundering via Child Task

作業上の親が、Spawnした子TaskのEgress Grantを承認できると、親自身の意図を子の通信として表現して自己承認する経路が生まれる。([権限モデルV2レビューEpisode](episode://design/authority-v2-review))

## 発生条件

- 親Agentが子Taskを生成できる。
- 子Taskの外向き通信がCASBでblockされる。
- 子のGrant Requestが作業上の親へ承認Escalationされる。
- ChallengeのTask identityやdelegation chainを見ない。

## 典型的な進行

1. 親が許可されていない外向き通信を望む。
2. 親がその処理を子Taskへ委譲する。
3. 子の通信がblockされ、Grantを要求する。
4. 要求が親へ上がる。
5. 親が「子の要求」として承認する。

## 起こりうる帰結

- Spawnによって実質権限が増える。
- 同じ目的の系統内で自己承認が成立する。
- 監査上のoriginが隠れる。

## 対策

- 作業階層とGovernance Planeを分離する。
- Policy Agentを作業Agentから独立させる。
- ChallengeへTask identityとdelegation chainを保存する。
- AuthorityをPolicyから解決する。
- CredentialをCredential Brokerだけが保持する。

## 適用限界

Sandbox内の実装方針やTask Contractの変更を親が判断すること自体はこのPatternに該当しない。問題は外向き通信のGrantを同じ親子系統で判断する場合である。

## 関連

- [Task Ownership](../schemas/task-ownership.md)
