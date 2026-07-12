use serde::{Deserialize, Serialize};
use serde_json::Value;
use thiserror::Error;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Envelope {
    pub message_id: String,
    pub message_type: String,
    pub from_component: String,
    pub to_component: String,
    pub correlation_id: String,
    #[serde(default)]
    pub causation_message_id: Option<String>,
    pub occurred_at: String,
    pub payload_schema: SchemaReference,
    pub payload: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SchemaReference {
    pub schema_id: String,
    pub schema_revision: u32,
    pub schema_digest: String,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum EnvelopeError {
    #[error("message identity is required")]
    MissingIdentity,
    #[error("distinct source and target components are required")]
    InvalidComponentRoute,
    #[error("correlation and timestamp are required")]
    MissingAuditMetadata,
    #[error("payload schema identity, revision, and digest are required")]
    MissingSchemaReference,
    #[error("payload must be a JSON object")]
    InvalidPayload,
}

impl Envelope {
    pub fn validate(&self) -> Result<(), EnvelopeError> {
        if self.message_id.is_empty() || self.message_type.is_empty() {
            return Err(EnvelopeError::MissingIdentity);
        }
        if self.from_component.is_empty()
            || self.to_component.is_empty()
            || self.from_component == self.to_component
        {
            return Err(EnvelopeError::InvalidComponentRoute);
        }
        if self.correlation_id.is_empty() || self.occurred_at.is_empty() {
            return Err(EnvelopeError::MissingAuditMetadata);
        }
        if self.payload_schema.schema_id.is_empty()
            || self.payload_schema.schema_revision == 0
            || self.payload_schema.schema_digest.is_empty()
        {
            return Err(EnvelopeError::MissingSchemaReference);
        }
        if !self.payload.is_object() {
            return Err(EnvelopeError::InvalidPayload);
        }
        Ok(())
    }
}
