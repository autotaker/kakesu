#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Decision {
    Allow,
    Block,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PolicyStoreState {
    Ready,
    Unavailable,
    Stale,
}

pub fn fail_closed_decision(store: PolicyStoreState, matched_allow_rule: bool) -> Decision {
    match (store, matched_allow_rule) {
        (PolicyStoreState::Ready, true) => Decision::Allow,
        _ => Decision::Block,
    }
}

#[cfg(test)]
mod tests {
    use super::{fail_closed_decision, Decision, PolicyStoreState};

    #[test]
    fn unavailable_policy_store_blocks() {
        assert_eq!(
            fail_closed_decision(PolicyStoreState::Unavailable, true),
            Decision::Block
        );
    }

    #[test]
    fn only_ready_explicit_allow_passes() {
        assert_eq!(
            fail_closed_decision(PolicyStoreState::Ready, true),
            Decision::Allow
        );
        assert_eq!(
            fail_closed_decision(PolicyStoreState::Ready, false),
            Decision::Block
        );
    }
}
