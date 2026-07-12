use kakesu_governance::message::{Envelope, SchemaReference};
use serde_json::json;

#[test]
fn governance_envelope_accepts_control_request() {
    let envelope = Envelope {
        message_id: "msg-1".into(),
        message_type: "GrantRequest".into(),
        from_component: "run-coordinator".into(),
        to_component: "governance-grant-service".into(),
        correlation_id: "task-1".into(),
        causation_message_id: None,
        occurred_at: "2026-07-13T00:00:00Z".into(),
        payload_schema: SchemaReference {
            schema_id: "urn:kakesu:governance-plane:grant-request:draft-v0:r1".into(),
            schema_revision: 1,
            schema_digest: "sha256:fixture".into(),
        },
        payload: json!({"challenge_id": "challenge-1"}),
    };

    assert!(envelope.validate().is_ok());
}
