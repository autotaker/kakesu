use kakesu_governance::wire_config::{load_wire_config, parse_wire_config};
use serde_json::{json, Value};
use std::fs;
use std::path::{Path, PathBuf};

fn project_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .to_path_buf()
}

#[test]
fn shared_wire_config_resolves_paths() {
    let root = project_root();
    let config = load_wire_config(&root).unwrap();
    assert_eq!(config.version, 1);
    assert_eq!(config.schema_catalog_root, root.join("schemas/draft-v0"));
    assert_eq!(config.components.len(), 3);
    let endpoints: std::collections::HashSet<_> = config
        .components
        .values()
        .map(|component| &component.uds_endpoint)
        .collect();
    assert_eq!(endpoints.len(), 3);
}

#[test]
fn wire_config_rejects_invalid_boundaries() {
    let root = project_root();
    let raw = fs::read(root.join("configs/local/wire.json")).unwrap();
    let original: Value = serde_json::from_slice(&raw).unwrap();
    for mutation in [
        "version",
        "missing",
        "extra",
        "absolute_catalog",
        "absolute_endpoint",
        "parent_endpoint",
        "empty_endpoint",
        "duplicate_endpoint",
    ] {
        let mut value = original.clone();
        match mutation {
            "version" => value["version"] = json!(2),
            "missing" => {
                value["components"]
                    .as_object_mut()
                    .unwrap()
                    .remove("core-runtime");
            }
            "extra" => value["components"]["extra"] = json!({"uds_endpoint": "run/extra.sock"}),
            "absolute_catalog" => value["schema_catalog_root"] = json!("/schemas"),
            "absolute_endpoint" => {
                value["components"]["core-runtime"]["uds_endpoint"] = json!("/run/core.sock")
            }
            "parent_endpoint" => {
                value["components"]["core-runtime"]["uds_endpoint"] = json!("run/../core.sock")
            }
            "empty_endpoint" => value["components"]["core-runtime"]["uds_endpoint"] = json!(""),
            _ => {
                value["components"]["core-runtime"]["uds_endpoint"] =
                    value["components"]["memory-service"]["uds_endpoint"].clone()
            }
        }
        assert!(
            parse_wire_config(&serde_json::to_vec(&value).unwrap(), &root).is_err(),
            "mutation {mutation} should be rejected"
        );
    }
}
