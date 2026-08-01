# コードレビュー

レビュアーはDEVと別Agentで、案 ブランチのproduct diffとDEVの`make check`証跡を独立に監査する。REVIEW_RESULTにはreviewer_agent、判断、reviewed_at、監査したDEV証跡、残存リスクを記録する。

レビューでは受け入れ条件、許可パス、テスト差分、失敗時の扱いを確認する。案の識別子はHANDOVERの`candidate_commit`だけから読み、レビュー結果へ重複転記しない。

P0/P1、検査未完了、受け入れ条件の根拠不足、または候補の製品差分だけという条件の不成立はPASSにしない。修正が必要な場合はDEVへ戻し、新しい案でREVIEWとQAを再開する。
