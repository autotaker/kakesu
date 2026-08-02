// Package approvalmanifest owns the canonical value representation of a push
// proposal. A valid Manifest is content to be considered for approval; it does
// not mean that the content is approved, granted, or pushable.
package approvalmanifest

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	// FormatVersion identifies the only wire format accepted by this package.
	FormatVersion = uint64(1)

	MaxEncodingBytes = 32 * 1024
	MaxRefUpdates    = 32

	maxRequestIDBytes     = 128
	maxAgentIDBytes       = 128
	maxWorkspaceIDBytes   = 128
	maxRepositoryBytes    = 256
	maxRemoteBytes        = 512
	maxPolicyVersionBytes = 128
	maxRefBytes           = 256
	objectIDBytes         = 40

	timestampLayout = "2006-01-02T15:04:05Z"
	digestLabel     = "sha256:"
	digestHexBytes  = 64
	digestDomain    = "dev-agent-harness/push-approval-manifest/v1\x00"
	zeroObjectID    = "0000000000000000000000000000000000000000"
)

// ErrorClass is a stable, non-sensitive error category.
type ErrorClass string

const (
	ClassInvalid      ErrorClass = "invalid"
	ClassParse        ErrorClass = "parse"
	ClassDuplicateKey ErrorClass = "duplicate_key"
	ClassDigest       ErrorClass = "digest"
	ClassNonCanonical ErrorClass = "non_canonical"
	ClassInternal     ErrorClass = "internal"
)

// Field is a stable location in a Proposal or its public encoding.
type Field string

const (
	FieldNone            Field = ""
	FieldEncoding        Field = "encoding"
	FieldFormatVersion   Field = "format_version"
	FieldRequestID       Field = "request_id"
	FieldAgentID         Field = "agent_id"
	FieldWorkspaceID     Field = "workspace_id"
	FieldRepository      Field = "repository"
	FieldRemote          Field = "remote"
	FieldRefUpdates      Field = "ref_updates"
	FieldRef             Field = "ref"
	FieldExpectedOldSHA  Field = "expected_old_sha"
	FieldNewSHA          Field = "new_sha"
	FieldForce           Field = "force"
	FieldDelete          Field = "delete"
	FieldPolicyVersion   Field = "policy_version"
	FieldRevocationEpoch Field = "revocation_epoch"
	FieldCreatedAt       Field = "created_at"
	FieldExpiresAt       Field = "expires_at"
	FieldRequestDigest   Field = "request_digest"
)

// Error exposes only a stable category and location. It intentionally does not
// wrap parser errors or include any caller-provided value.
type Error struct {
	class ErrorClass
	field Field
	index int
}

func (e *Error) Error() string {
	if e == nil {
		return "approval manifest error"
	}
	if e.index >= 0 {
		return "approval manifest " + string(e.class) + " at " + string(e.field) + " in ref update"
	}
	if e.field != FieldNone {
		return "approval manifest " + string(e.class) + " at " + string(e.field)
	}
	return "approval manifest " + string(e.class)
}

// Class returns the stable category of e.
func (e *Error) Class() ErrorClass {
	if e == nil {
		return ""
	}
	return e.class
}

// Field returns the stable field location of e.
func (e *Error) Field() Field {
	if e == nil {
		return FieldNone
	}
	return e.field
}

// RefUpdateIndex returns the failing ref-update index and true when the error
// refers to an update. It never returns caller data.
func (e *Error) RefUpdateIndex() (int, bool) {
	if e == nil || e.index < 0 {
		return 0, false
	}
	return e.index, true
}

// ClassOf returns a stable class for package errors and ClassInternal for any
// unexpected error. No lower-level error text is exposed.
func ClassOf(err error) ErrorClass {
	var target *Error
	if errors.As(err, &target) {
		return target.class
	}
	if err == nil {
		return ""
	}
	return ClassInternal
}

// LocationOf returns the stable field and optional ref-update index.
func LocationOf(err error) (Field, int, bool) {
	var target *Error
	if !errors.As(err, &target) {
		return FieldNone, 0, false
	}
	index, indexed := target.RefUpdateIndex()
	return target.field, index, indexed
}

func newError(class ErrorClass, field Field) error {
	return &Error{class: class, field: field, index: -1}
}

func newRefError(field Field, index int) error {
	return &Error{class: ClassInvalid, field: field, index: index}
}

func newIndexedError(class ErrorClass, field Field, index int) error {
	return &Error{class: class, field: field, index: index}
}

// RefUpdate describes one caller-observed branch update. Force is explicit and
// is included in the digest; the package does not infer it from commit history.
type RefUpdate struct {
	Ref            string
	ExpectedOldSHA string
	NewSHA         string
	Force          bool
	Delete         bool
}

// Proposal contains every value bound into a v1 manifest. The package does not
// generate or authorize any of these values.
type Proposal struct {
	RequestID       string
	AgentID         string
	WorkspaceID     string
	Repository      string
	Remote          string
	RefUpdates      []RefUpdate
	PolicyVersion   string
	RevocationEpoch uint64
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

// Manifest is immutable to callers. All mutable getters return fresh copies.
type Manifest struct {
	formatVersion   uint64
	requestID       string
	agentID         string
	workspaceID     string
	repository      string
	remote          string
	refUpdates      []RefUpdate
	policyVersion   string
	revocationEpoch uint64
	createdAt       time.Time
	expiresAt       time.Time
	requestDigest   string
	encoding        []byte
}

// Build validates a complete proposal and returns its canonical representation.
func Build(proposal Proposal) (*Manifest, error) {
	if err := validateProposal(proposal); err != nil {
		return nil, err
	}
	return buildValidated(proposal)
}

// Parse accepts only the exact byte representation emitted by Build.
func Parse(raw []byte) (*Manifest, error) {
	if len(raw) == 0 || len(raw) > MaxEncodingBytes {
		return nil, newError(ClassParse, FieldEncoding)
	}
	if err := scanSingleJSONDocument(raw); err != nil {
		return nil, err
	}
	if err := requireWireFields(raw); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var wire publicWire
	if err := decoder.Decode(&wire); err != nil {
		return nil, newError(ClassParse, FieldEncoding)
	}
	if err := requireDecoderEOF(decoder); err != nil {
		return nil, err
	}
	if wire.FormatVersion != FormatVersion {
		return nil, newError(ClassInvalid, FieldFormatVersion)
	}

	proposal := proposalFromWire(wire)
	if err := validateProposal(proposal); err != nil {
		return nil, err
	}
	if !validDigestSpelling(wire.RequestDigest) {
		return nil, newError(ClassDigest, FieldRequestDigest)
	}

	manifest, err := buildValidated(proposal)
	if err != nil {
		return nil, newError(ClassInternal, FieldEncoding)
	}
	if subtle.ConstantTimeCompare([]byte(wire.RequestDigest), []byte(manifest.requestDigest)) != 1 {
		return nil, newError(ClassDigest, FieldRequestDigest)
	}
	if !bytes.Equal(raw, manifest.encoding) {
		return nil, newError(ClassNonCanonical, FieldEncoding)
	}
	return manifest, nil
}

func buildValidated(proposal Proposal) (*Manifest, error) {
	payload := payloadFromProposal(proposal)
	payloadEncoding, err := json.Marshal(payload)
	if err != nil {
		return nil, newError(ClassInternal, FieldEncoding)
	}

	hasherInput := make([]byte, 0, len(digestDomain)+len(payloadEncoding))
	hasherInput = append(hasherInput, digestDomain...)
	hasherInput = append(hasherInput, payloadEncoding...)
	digestBytes := sha256.Sum256(hasherInput)
	digest := digestLabel + hex.EncodeToString(digestBytes[:])

	public := publicWire{
		FormatVersion:   payload.FormatVersion,
		RequestID:       payload.RequestID,
		AgentID:         payload.AgentID,
		WorkspaceID:     payload.WorkspaceID,
		Repository:      payload.Repository,
		Remote:          payload.Remote,
		RefUpdates:      payload.RefUpdates,
		PolicyVersion:   payload.PolicyVersion,
		RevocationEpoch: payload.RevocationEpoch,
		CreatedAt:       payload.CreatedAt,
		ExpiresAt:       payload.ExpiresAt,
		RequestDigest:   digest,
	}
	publicEncoding, err := json.Marshal(public)
	if err != nil || len(publicEncoding) > MaxEncodingBytes {
		return nil, newError(ClassInternal, FieldEncoding)
	}

	return &Manifest{
		formatVersion:   FormatVersion,
		requestID:       proposal.RequestID,
		agentID:         proposal.AgentID,
		workspaceID:     proposal.WorkspaceID,
		repository:      proposal.Repository,
		remote:          proposal.Remote,
		refUpdates:      cloneRefUpdates(proposal.RefUpdates),
		policyVersion:   proposal.PolicyVersion,
		revocationEpoch: proposal.RevocationEpoch,
		createdAt:       proposal.CreatedAt,
		expiresAt:       proposal.ExpiresAt,
		requestDigest:   digest,
		encoding:        append([]byte(nil), publicEncoding...),
	}, nil
}

type refUpdateWire struct {
	Ref            string `json:"ref"`
	ExpectedOldSHA string `json:"expected_old_sha"`
	NewSHA         string `json:"new_sha"`
	Force          bool   `json:"force"`
	Delete         bool   `json:"delete"`
}

type payloadWire struct {
	FormatVersion   uint64          `json:"format_version"`
	RequestID       string          `json:"request_id"`
	AgentID         string          `json:"agent_id"`
	WorkspaceID     string          `json:"workspace_id"`
	Repository      string          `json:"repository"`
	Remote          string          `json:"remote"`
	RefUpdates      []refUpdateWire `json:"ref_updates"`
	PolicyVersion   string          `json:"policy_version"`
	RevocationEpoch uint64          `json:"revocation_epoch"`
	CreatedAt       string          `json:"created_at"`
	ExpiresAt       string          `json:"expires_at"`
}

type publicWire struct {
	FormatVersion   uint64          `json:"format_version"`
	RequestID       string          `json:"request_id"`
	AgentID         string          `json:"agent_id"`
	WorkspaceID     string          `json:"workspace_id"`
	Repository      string          `json:"repository"`
	Remote          string          `json:"remote"`
	RefUpdates      []refUpdateWire `json:"ref_updates"`
	PolicyVersion   string          `json:"policy_version"`
	RevocationEpoch uint64          `json:"revocation_epoch"`
	CreatedAt       string          `json:"created_at"`
	ExpiresAt       string          `json:"expires_at"`
	RequestDigest   string          `json:"request_digest"`
}

func payloadFromProposal(proposal Proposal) payloadWire {
	updates := make([]refUpdateWire, len(proposal.RefUpdates))
	for index, update := range proposal.RefUpdates {
		updates[index] = refUpdateWire{
			Ref:            update.Ref,
			ExpectedOldSHA: update.ExpectedOldSHA,
			NewSHA:         update.NewSHA,
			Force:          update.Force,
			Delete:         update.Delete,
		}
	}
	return payloadWire{
		FormatVersion:   FormatVersion,
		RequestID:       proposal.RequestID,
		AgentID:         proposal.AgentID,
		WorkspaceID:     proposal.WorkspaceID,
		Repository:      proposal.Repository,
		Remote:          proposal.Remote,
		RefUpdates:      updates,
		PolicyVersion:   proposal.PolicyVersion,
		RevocationEpoch: proposal.RevocationEpoch,
		CreatedAt:       proposal.CreatedAt.Format(timestampLayout),
		ExpiresAt:       proposal.ExpiresAt.Format(timestampLayout),
	}
}

func proposalFromWire(wire publicWire) Proposal {
	updates := make([]RefUpdate, len(wire.RefUpdates))
	for index, update := range wire.RefUpdates {
		updates[index] = RefUpdate{
			Ref:            update.Ref,
			ExpectedOldSHA: update.ExpectedOldSHA,
			NewSHA:         update.NewSHA,
			Force:          update.Force,
			Delete:         update.Delete,
		}
	}
	createdAt, _ := time.Parse(timestampLayout, wire.CreatedAt)
	expiresAt, _ := time.Parse(timestampLayout, wire.ExpiresAt)
	return Proposal{
		RequestID:       wire.RequestID,
		AgentID:         wire.AgentID,
		WorkspaceID:     wire.WorkspaceID,
		Repository:      wire.Repository,
		Remote:          wire.Remote,
		RefUpdates:      updates,
		PolicyVersion:   wire.PolicyVersion,
		RevocationEpoch: wire.RevocationEpoch,
		CreatedAt:       createdAt,
		ExpiresAt:       expiresAt,
	}
}

func validateProposal(proposal Proposal) error {
	checks := []struct {
		value string
		limit int
		field Field
	}{
		{proposal.RequestID, maxRequestIDBytes, FieldRequestID},
		{proposal.AgentID, maxAgentIDBytes, FieldAgentID},
		{proposal.WorkspaceID, maxWorkspaceIDBytes, FieldWorkspaceID},
		{proposal.PolicyVersion, maxPolicyVersionBytes, FieldPolicyVersion},
	}
	for _, check := range checks {
		if !validToken(check.value, check.limit) {
			return newError(ClassInvalid, check.field)
		}
	}
	if !validRepository(proposal.Repository) {
		return newError(ClassInvalid, FieldRepository)
	}
	expectedRemote := "https://github.com/" + proposal.Repository + ".git"
	if len(proposal.Remote) > maxRemoteBytes || proposal.Remote != expectedRemote {
		return newError(ClassInvalid, FieldRemote)
	}
	if len(proposal.RefUpdates) < 1 || len(proposal.RefUpdates) > MaxRefUpdates {
		return newError(ClassInvalid, FieldRefUpdates)
	}
	seen := make(map[string]struct{}, len(proposal.RefUpdates))
	for index, update := range proposal.RefUpdates {
		if !validBranchRef(update.Ref) {
			return newRefError(FieldRef, index)
		}
		if _, duplicate := seen[update.Ref]; duplicate {
			return newRefError(FieldRef, index)
		}
		seen[update.Ref] = struct{}{}
		if !validObjectID(update.ExpectedOldSHA) {
			return newRefError(FieldExpectedOldSHA, index)
		}
		if !validObjectID(update.NewSHA) {
			return newRefError(FieldNewSHA, index)
		}
		if field := invalidTransitionField(update); field != FieldNone {
			return newRefError(field, index)
		}
	}
	if !validTime(proposal.CreatedAt) {
		return newError(ClassInvalid, FieldCreatedAt)
	}
	if !validTime(proposal.ExpiresAt) {
		return newError(ClassInvalid, FieldExpiresAt)
	}
	if !proposal.CreatedAt.Before(proposal.ExpiresAt) {
		return newError(ClassInvalid, FieldExpiresAt)
	}
	return nil
}

func validToken(value string, limit int) bool {
	if len(value) == 0 || len(value) > limit {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if isASCIIAlphaNumeric(character) || strings.ContainsRune("._:@/-", rune(character)) {
			continue
		}
		return false
	}
	first := value[0]
	last := value[len(value)-1]
	return isASCIIAlphaNumeric(first) && isASCIIAlphaNumeric(last)
}

func validRepository(repository string) bool {
	if len(repository) == 0 || len(repository) > maxRepositoryBytes || repository != strings.ToLower(repository) {
		return false
	}
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || !validRepositoryPart(parts[0]) || !validRepositoryPart(parts[1]) {
		return false
	}
	return !strings.HasSuffix(parts[1], ".git")
}

func validRepositoryPart(part string) bool {
	if len(part) == 0 || !isASCIILowerNumeric(part[0]) || part[len(part)-1] == '.' {
		return false
	}
	for index := 0; index < len(part); index++ {
		character := part[index]
		if isASCIILowerNumeric(character) || character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func validBranchRef(ref string) bool {
	const prefix = "refs/heads/"
	if len(ref) > maxRefBytes || !strings.HasPrefix(ref, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(ref, prefix)
	if suffix == "" || strings.Contains(suffix, "..") || strings.HasPrefix(suffix, "/") || strings.HasSuffix(suffix, "/") {
		return false
	}
	for _, segment := range strings.Split(suffix, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasPrefix(segment, ".") || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, ".lock") {
			return false
		}
		for index := 0; index < len(segment); index++ {
			character := segment[index]
			if isASCIIAlphaNumeric(character) || character == '-' || character == '_' || character == '.' {
				continue
			}
			return false
		}
	}
	return true
}

func validObjectID(objectID string) bool {
	if len(objectID) != objectIDBytes {
		return false
	}
	for index := 0; index < len(objectID); index++ {
		character := objectID[index]
		if (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') {
			continue
		}
		return false
	}
	return true
}

func invalidTransitionField(update RefUpdate) Field {
	oldIsZero := update.ExpectedOldSHA == zeroObjectID
	newIsZero := update.NewSHA == zeroObjectID
	switch {
	case oldIsZero && !newIsZero:
		if update.Force {
			return FieldForce
		}
		if update.Delete {
			return FieldDelete
		}
		return FieldNone
	case !oldIsZero && newIsZero:
		if update.Force {
			return FieldForce
		}
		if !update.Delete {
			return FieldDelete
		}
		return FieldNone
	case !oldIsZero && !newIsZero:
		if update.ExpectedOldSHA == update.NewSHA {
			return FieldNewSHA
		}
		if update.Delete {
			return FieldDelete
		}
		return FieldNone
	default:
		return FieldNewSHA
	}
}

func validTime(value time.Time) bool {
	if value.IsZero() || value.Location() != time.UTC || value.Nanosecond() != 0 {
		return false
	}
	encoded := value.Format(timestampLayout)
	parsed, err := time.Parse(timestampLayout, encoded)
	return err == nil && parsed.Equal(value) && encoded == value.Format(time.RFC3339)
}

func validDigestSpelling(digest string) bool {
	if len(digest) != len(digestLabel)+digestHexBytes || !strings.HasPrefix(digest, digestLabel) {
		return false
	}
	for index := len(digestLabel); index < len(digest); index++ {
		character := digest[index]
		if (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') {
			continue
		}
		return false
	}
	return true
}

func isASCIIAlphaNumeric(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
}

func isASCIILowerNumeric(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9')
}

func scanSingleJSONDocument(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	return requireDecoderEOF(decoder)
}

func requireWireFields(raw []byte) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return newError(ClassParse, FieldEncoding)
	}
	required := []struct {
		name  string
		field Field
	}{
		{"format_version", FieldFormatVersion},
		{"request_id", FieldRequestID},
		{"agent_id", FieldAgentID},
		{"workspace_id", FieldWorkspaceID},
		{"repository", FieldRepository},
		{"remote", FieldRemote},
		{"ref_updates", FieldRefUpdates},
		{"policy_version", FieldPolicyVersion},
		{"revocation_epoch", FieldRevocationEpoch},
		{"created_at", FieldCreatedAt},
		{"expires_at", FieldExpiresAt},
		{"request_digest", FieldRequestDigest},
	}
	for _, entry := range required {
		if _, present := object[entry.name]; !present {
			return newError(ClassParse, entry.field)
		}
	}
	var updates []map[string]json.RawMessage
	if err := json.Unmarshal(object["ref_updates"], &updates); err != nil {
		return newError(ClassParse, FieldRefUpdates)
	}
	for index, update := range updates {
		for _, field := range []struct {
			name     string
			location Field
		}{
			{"ref", FieldRef},
			{"expected_old_sha", FieldExpectedOldSHA},
			{"new_sha", FieldNewSHA},
			{"force", FieldForce},
			{"delete", FieldDelete},
		} {
			if _, present := update[field.name]; !present {
				return newIndexedError(ClassParse, field.location, index)
			}
		}
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return newError(ClassParse, FieldEncoding)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return newError(ClassParse, FieldEncoding)
			}
			key, ok := keyToken.(string)
			if !ok {
				return newError(ClassParse, FieldEncoding)
			}
			if _, duplicate := seen[key]; duplicate {
				return newError(ClassDuplicateKey, FieldEncoding)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return newError(ClassParse, FieldEncoding)
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return newError(ClassParse, FieldEncoding)
		}
	default:
		return newError(ClassParse, FieldEncoding)
	}
	return nil
}

func requireDecoderEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return newError(ClassParse, FieldEncoding)
	}
	return nil
}

func cloneRefUpdates(updates []RefUpdate) []RefUpdate {
	return append([]RefUpdate(nil), updates...)
}

func (manifest *Manifest) FormatVersion() uint64 { return manifest.formatVersion }

func (manifest *Manifest) RequestID() string { return manifest.requestID }

func (manifest *Manifest) AgentID() string { return manifest.agentID }

func (manifest *Manifest) WorkspaceID() string { return manifest.workspaceID }

func (manifest *Manifest) Repository() string { return manifest.repository }

func (manifest *Manifest) Remote() string { return manifest.remote }

func (manifest *Manifest) RefUpdates() []RefUpdate { return cloneRefUpdates(manifest.refUpdates) }

func (manifest *Manifest) PolicyVersion() string { return manifest.policyVersion }

func (manifest *Manifest) RevocationEpoch() uint64 { return manifest.revocationEpoch }

func (manifest *Manifest) CreatedAt() time.Time { return manifest.createdAt }

func (manifest *Manifest) ExpiresAt() time.Time { return manifest.expiresAt }

func (manifest *Manifest) Digest() string { return manifest.requestDigest }

func (manifest *Manifest) Encoding() []byte {
	return append([]byte(nil), manifest.encoding...)
}
