---
kind: schema
title: Development Agent Harness Provision Manifest
---

# Development Agent Harness Provision Manifest

## 問い

Development Agent Harness がOS上の配置を実行せずに、再現可能で検証可能な provision plan として表現するにはどうするか。

## Manifest V1 の生成契約

`dev-agent-harness-setup plan-provision --config PATH --target-root PATH`は、対象OSのuser、directory、serviceの望ましい状態を出力する。V1はheader一行の後にaction十行を置く決定的JSONLであり、fieldとrecord順序はexact-bytes testで固定する。互換性を伴う拡張は既存versionを曖昧に維持せず、別Taskでversionを定義する。

全recordはwriterを呼ぶ前に構築、検証、serializeする。出力adapterは一回だけwriteし、retryやre-emitをしない。このcommandはtarget rootとhostを変更せず、process、network、IPCも開始しない。executor、install、deploy、および設定生成はこの境界の外にある。

## Target mapping の座標境界

logical pathと`target-root`は別のcoordinateとして扱う。logical `/`だけは空の相対成分となるため拒否するが、target rootと同じ文字列であること自体は拒否理由ではない。empty、relative、cleanでないpath、NULを含むpath、root logical pathは、mappingの前にrejectする。

この区別により、論理的な配置先の安全な検証と、テストまたは隔離先へのtarget-root mappingを混同しない。

## 部分writeの限界

OS-level writerがprefixを書いた後にerrorを返したとき、既に書かれたbyteはapplication側から巻き戻せない。したがってこの契約の保証は、事前全件検証、single write、retry/re-emitなしまでである。writer自身のatomicityやOS-level rollbackを保証するものではない。

## 関連

- [TASK-0036 HANDOVER](../../../tasks/TASK-0036-dev-agent-provision-manifest/HANDOVER.md)
