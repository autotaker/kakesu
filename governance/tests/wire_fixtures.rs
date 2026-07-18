use kakesu_governance::message::decode_and_validate;
use serde_json::Value;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

fn project_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .parent()
        .unwrap()
        .to_path_buf()
}

#[test]
fn shared_envelope_fixture_matrix() {
    let root = project_root();
    let catalog = root.join("schemas/draft-v0");
    for (fixture, valid) in [
        ("valid.json", true),
        ("invalid-same-route.json", false),
        ("invalid-non-utc-timestamp.json", false),
        ("invalid-schema-reference-digest.json", false),
        ("invalid-non-object-payload.json", false),
    ] {
        let raw = fs::read(root.join("fixtures/wire/envelope").join(fixture)).unwrap();
        assert_eq!(
            decode_and_validate(&raw, &catalog).is_ok(),
            valid,
            "unexpected result for {fixture}"
        );
    }
}

#[test]
fn envelope_requires_exact_fields_and_explicit_null() {
    let root = project_root();
    let catalog = root.join("schemas/draft-v0");
    let raw = fs::read(root.join("fixtures/wire/envelope/valid.json")).unwrap();
    let mut value: Value = serde_json::from_slice(&raw).unwrap();
    value
        .as_object_mut()
        .unwrap()
        .remove("causation_message_id");
    assert!(decode_and_validate(&serde_json::to_vec(&value).unwrap(), &catalog).is_err());
    value["causation_message_id"] = Value::Null;
    value["unknown"] = Value::Bool(true);
    assert!(decode_and_validate(&serde_json::to_vec(&value).unwrap(), &catalog).is_err());
}

#[test]
fn envelope_rejects_raw_schema_mutation() {
    let root = project_root();
    let raw = fs::read(root.join("fixtures/wire/envelope/valid.json")).unwrap();
    let mut schema = fs::read(root.join("schemas/draft-v0/common/error.schema.json")).unwrap();
    schema.push(b' ');
    let nonce = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    let catalog =
        std::env::temp_dir().join(format!("kakesu-schema-{}-{nonce}", std::process::id()));
    fs::create_dir(&catalog).unwrap();
    fs::write(catalog.join("error.schema.json"), schema).unwrap();
    let result = decode_and_validate(&raw, &catalog);
    fs::remove_dir_all(&catalog).unwrap();
    assert!(result.is_err());
}
