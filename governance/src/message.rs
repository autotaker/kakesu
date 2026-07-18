use chrono::DateTime;
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::fs;
use std::path::{Path, PathBuf};
use thiserror::Error;

#[derive(Debug, Clone, Serialize, PartialEq)]
pub struct Envelope {
    pub message_id: String,
    pub message_type: String,
    pub from_component: String,
    pub to_component: String,
    pub correlation_id: String,
    pub causation_message_id: Option<String>,
    pub occurred_at: String,
    pub payload_schema: SchemaReference,
    pub payload: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct SchemaReference {
    pub schema_id: String,
    pub schema_revision: u32,
    pub schema_digest: String,
}

#[derive(Deserialize)]
struct RequiredNullable(Option<String>);

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct WireEnvelope {
    message_id: String,
    message_type: String,
    from_component: String,
    to_component: String,
    correlation_id: String,
    causation_message_id: RequiredNullable,
    occurred_at: String,
    payload_schema: SchemaReference,
    payload: Value,
}

impl<'de> Deserialize<'de> for Envelope {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let wire = WireEnvelope::deserialize(deserializer)?;
        Ok(Self {
            message_id: wire.message_id,
            message_type: wire.message_type,
            from_component: wire.from_component,
            to_component: wire.to_component,
            correlation_id: wire.correlation_id,
            causation_message_id: wire.causation_message_id.0,
            occurred_at: wire.occurred_at,
            payload_schema: wire.payload_schema,
            payload: wire.payload,
        })
    }
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum EnvelopeError {
    #[error("message identity is required")]
    MissingIdentity,
    #[error("distinct source and target components are required")]
    InvalidComponentRoute,
    #[error("correlation and timestamp are required")]
    MissingAuditMetadata,
    #[error("occurred_at must be RFC3339 UTC with a trailing Z")]
    InvalidTimestamp,
    #[error("payload schema identity, revision, and digest are required")]
    MissingSchemaReference,
    #[error("payload must be a JSON object")]
    InvalidPayload,
    #[error("invalid envelope JSON: {0}")]
    Decode(String),
    #[error("schema catalog error: {0}")]
    Catalog(String),
    #[error("schema digest does not match catalog artifact bytes")]
    SchemaDigestMismatch,
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
        if !self.occurred_at.ends_with('Z')
            || DateTime::parse_from_rfc3339(&self.occurred_at)
                .map(|timestamp| timestamp.offset().local_minus_utc() != 0)
                .unwrap_or(true)
        {
            return Err(EnvelopeError::InvalidTimestamp);
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

pub fn decode_and_validate(data: &[u8], catalog_root: &Path) -> Result<Envelope, EnvelopeError> {
    let envelope: Envelope =
        serde_json::from_slice(data).map_err(|error| EnvelopeError::Decode(error.to_string()))?;
    envelope.validate()?;
    validate_schema_reference(&envelope.payload_schema, catalog_root)?;
    Ok(envelope)
}

fn validate_schema_reference(
    reference: &SchemaReference,
    catalog_root: &Path,
) -> Result<(), EnvelopeError> {
    if !valid_digest(&reference.schema_digest) {
        return Err(EnvelopeError::Catalog(
            "schema_digest must be sha256 followed by lowercase hex".into(),
        ));
    }
    let mut paths = Vec::new();
    collect_json_paths(catalog_root, &mut paths)?;
    let mut matches = Vec::new();
    for path in paths {
        let raw = fs::read(&path).map_err(|error| EnvelopeError::Catalog(error.to_string()))?;
        let metadata: Value = serde_json::from_slice(&raw)
            .map_err(|error| EnvelopeError::Catalog(format!("{}: {error}", path.display())))?;
        if metadata.get("$id").and_then(Value::as_str) == Some(reference.schema_id.as_str())
            && metadata.get("x-schema-revision").and_then(Value::as_u64)
                == Some(u64::from(reference.schema_revision))
        {
            matches.push(raw);
        }
    }
    if matches.len() != 1 {
        return Err(EnvelopeError::Catalog(format!(
            "schema reference resolved {} times",
            matches.len()
        )));
    }
    let digest = format!("sha256:{:x}", Sha256::digest(&matches[0]));
    if digest != reference.schema_digest {
        return Err(EnvelopeError::SchemaDigestMismatch);
    }
    Ok(())
}

fn collect_json_paths(root: &Path, paths: &mut Vec<PathBuf>) -> Result<(), EnvelopeError> {
    let entries = fs::read_dir(root).map_err(|error| EnvelopeError::Catalog(error.to_string()))?;
    for entry in entries {
        let entry = entry.map_err(|error| EnvelopeError::Catalog(error.to_string()))?;
        let path = entry.path();
        if path.is_dir() {
            collect_json_paths(&path, paths)?;
        } else if path.extension().and_then(|extension| extension.to_str()) == Some("json") {
            paths.push(path);
        }
    }
    Ok(())
}

fn valid_digest(value: &str) -> bool {
    value.len() == 71
        && value.starts_with("sha256:")
        && value[7..]
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}
