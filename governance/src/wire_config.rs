use serde::Deserialize;
use std::collections::{HashMap, HashSet};
use std::fs;
use std::path::{Component, Path, PathBuf};
use thiserror::Error;

const COMPONENTS: [&str; 3] = ["core-runtime", "memory-service", "governance-service"];

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WireConfig {
    pub version: u32,
    pub schema_catalog_root: PathBuf,
    pub components: HashMap<String, ComponentConfig>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ComponentConfig {
    pub uds_endpoint: PathBuf,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawWireConfig {
    version: u32,
    schema_catalog_root: String,
    components: HashMap<String, RawComponentConfig>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RawComponentConfig {
    uds_endpoint: String,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum WireConfigError {
    #[error("unable to read wire config: {0}")]
    Io(String),
    #[error("invalid wire config JSON: {0}")]
    Decode(String),
    #[error("wire config version must be 1")]
    Version,
    #[error("wire config must contain exactly the three known components")]
    Components,
    #[error("wire config paths must be non-empty, safe, and relative")]
    UnsafePath,
    #[error("UDS endpoints must be unique")]
    DuplicateEndpoint,
}

pub fn load_wire_config(project_root: &Path) -> Result<WireConfig, WireConfigError> {
    let raw = fs::read(project_root.join("configs/local/wire.json"))
        .map_err(|error| WireConfigError::Io(error.to_string()))?;
    parse_wire_config(&raw, project_root)
}

pub fn parse_wire_config(data: &[u8], project_root: &Path) -> Result<WireConfig, WireConfigError> {
    let raw: RawWireConfig =
        serde_json::from_slice(data).map_err(|error| WireConfigError::Decode(error.to_string()))?;
    if raw.version != 1 {
        return Err(WireConfigError::Version);
    }
    let catalog = safe_relative(&raw.schema_catalog_root)?;
    if raw.components.len() != COMPONENTS.len()
        || !COMPONENTS
            .iter()
            .all(|name| raw.components.contains_key(*name))
    {
        return Err(WireConfigError::Components);
    }
    let mut endpoints = HashSet::new();
    let mut components = HashMap::new();
    for name in COMPONENTS {
        let endpoint = safe_relative(&raw.components[name].uds_endpoint)?;
        if !endpoints.insert(endpoint.clone()) {
            return Err(WireConfigError::DuplicateEndpoint);
        }
        components.insert(
            name.to_owned(),
            ComponentConfig {
                uds_endpoint: project_root.join(endpoint),
            },
        );
    }
    Ok(WireConfig {
        version: 1,
        schema_catalog_root: project_root.join(catalog),
        components,
    })
}

fn safe_relative(value: &str) -> Result<PathBuf, WireConfigError> {
    let path = Path::new(value);
    if value.is_empty()
        || path.is_absolute()
        || path == Path::new(".")
        || path
            .components()
            .any(|component| matches!(component, Component::ParentDir))
    {
        return Err(WireConfigError::UnsafePath);
    }
    Ok(path.to_path_buf())
}
