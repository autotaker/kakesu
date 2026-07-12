// Canonical logical types for the design. Persistence-specific fields may be added by implementations.

export type AgentLevel = "L1" | "L2" | "L3";

export interface Agent {
  agentId: string;
  profileId: string;
  level: AgentLevel;
  status: "idle" | "assigned" | "retired";
  currentTaskId?: string;
}

export type TaskStatus =
  | "ready"
  | "running"
  | "waiting"
  | "suspended"
  | "reviewing_completion"
  | "completed"
  | "cancelled";

export interface TaskContract {
  objective: string;
  acceptance: string;
  instructions?: string;
  version: number;
}

export interface Task {
  taskId: string;
  parentTaskId?: string;
  ownerAgentId: string;
  workspaceId: string;
  contract: TaskContract;
  status: TaskStatus;
  dependency: "required" | "optional";
  version: number;
  createdAt: string;
  startedAt?: string;
  endedAt?: string;
}

export interface TaskEvent {
  eventId: string;
  taskId: string;
  actorKind: "work_agent" | "harness" | "authority" | "internal_component";
  actorRef: string;
  eventType: string;
  payloadRef: string;
  createdAt: string;
}

export interface Workspace {
  workspaceId: string;
  taskId: string;
  sourceWorkspaceId?: string;
  mode: "fork" | "shared_readonly" | "empty";
  storageRef: string;
  status: "active" | "frozen" | "archived" | "destroyed";
}

export interface AgentRun {
  runId: string;
  agentId: string;
  taskId: string;
  status: "running" | "stopped" | "failed" | "completed";
  previousResponseId?: string;
  continuationId?: string;
  resumeCursorId?: string;
  stopReason?: "waiting" | "compacted" | "shutdown";
  normalStepCount: number;
  lastProgressRefreshStep: number;
  startedAt: string;
  endedAt?: string;
}

export interface AgentRunStep {
  stepId: string;
  runId: string;
  sequence: number;
  assignmentEventSequence: number;
  kind: "normal" | "progress_maintenance";
  responseId?: string;
  status: "running" | "completed" | "failed";
  requestContextDigest: string;
  contractVersion: number;
  taskVersion: number;
  progressVersion: number;
  startedAt: string;
  endedAt?: string;
}

export interface AgentRunItem {
  itemId: string;
  stepId: string;
  outputIndex: number;
  type: string;
  status?: string;
  rawDigest: string;
  contentRef?: string;
  retentionClass: "long" | "policy" | "short";
}

export interface AgentResource {
  resourceId: string;
  agentId: string;
  assignmentId?: string;
  runId?: string;
  kind: "process" | "server" | "worktree" | "temporary_directory";
  resourceRef: string;
  lifetime: "run" | "assignment" | "agent";
  cleanupPolicy: "stop" | "delete" | "retain";
  status:
    | "active"
    | "cleanup_pending"
    | "cleaning"
    | "released"
    | "needs_operator";
}

export interface ResumeCursor {
  cursorId: string;
  taskId: string;
  agentId: string;
  sourceRunId: string;
  contractVersion: number;
  taskVersion: number;
  progressVersion: number;
  workspaceSnapshotRef: string;
  lastConsumedMailboxSequence: number;
  lastObservedTaskEventSequence: number;
  lastObservedAgentRunEventSequence: number;
  createdAt: string;
}

export interface TaskProgress {
  taskId: string;
  version: number;
  currentFocusId?: string;
  lastObservedTaskEventSequence: number;
  lastObservedAgentRunEventSequence: number;
  items: ProgressItem[];
  updatedAt: string;
}

export interface ProgressItem {
  itemId: string;
  description: string;
  status: "pending" | "in_progress" | "completed" | "blocked" | "cancelled";
  evidenceRefs: string[];
  blocker?: string;
}

export interface AgentRunEventView {
  assignmentEventSequence: number;
  runId: string;
  stepId: string;
  kind: "message" | "function_call" | "function_result" | "error" | "maintenance";
  summary?: string;
  ref?: string;
  occurredAt: string;
}

export interface Continuation {
  continuationId: string;
  taskId: string;
  runId: string;
  reason: "waiting" | "reviewing_completion" | "suspended";
  waitCondition?: WaitCondition;
  awaitedEventIds: string[];
  previousResponseId?: string;
  pendingCallId?: string;
  contractVersion: number;
  workspaceSnapshotRef: string;
  contextSnapshotRef: string;
}

export type WaitCondition =
  | { kind: "child"; asyncIds: string[]; mode: "all" | "any" }
  | { kind: "parent"; requestId: string }
  | { kind: "grant"; asyncId: string }
  | { kind: "timer"; wakeAt: string };

export type ParentRequestStatus = "pending" | "resolved" | "cancelled";

export interface AskRequest {
  requestId: string;
  childTaskId: string;
  parentTaskId: string;
  childOwnerAgentId: string;
  parentOwnerAgentId: string;
  contractVersion: number;
  question: string;
  status: ParentRequestStatus;
  asyncId: string;
}

export interface AskAdvice {
  requestId: string;
  advice: string;
  responderAgentId: string;
  resolvedAt: string;
}

export interface EscalationRequest {
  requestId: string;
  requesterTaskId: string;
  authorityTaskId?: string;
  rootAuthorityRef?: string;
  contractVersion: number;
  question: string;
  proposedOptions: string[];
  status: ParentRequestStatus;
  asyncId: string;
}

export interface EscalationDecision {
  requestId: string;
  authorityRef: string;
  decision: string;
  contractPatch?: Partial<TaskContract>;
  terminate: boolean;
  resolvedAt: string;
}

export interface Suspension {
  reason: string;
  source:
    | "agent_run_failure"
    | "runtime_failure"
    | "workspace_failure"
    | "resource_unavailable"
    | "built_in_job_failure";
  recoveryOwner: "harness" | "operator";
  recoveryPolicy: "automatic" | "manual";
  retryCount: number;
  suspendedAt: string;
  nextRetryAt?: string;
}

export type ToolResult<T> =
  | { status: "completed"; value: T }
  | { status: "accepted"; asyncId: string; operation: string }
  | { status: "failed"; error: ToolError };

export interface ToolError {
  code: string;
  message: string;
  retryable: boolean;
  detailsRef?: string;
}

export interface AsyncOperation {
  asyncId: string;
  ownerTaskId: string;
  toolCallId?: string;
  toolName: string;
  status: "running" | "completed" | "failed" | "cancelled";
  startedAt: string;
  syncDeadline?: string;
  completedAt?: string;
  resultRef?: string;
  errorRef?: string;
  operationKey: string;
}

export type EgressMailboxEvent =
  | {
      eventId: string;
      type: "EgressBlocked";
      workspaceId: string;
      taskId: string;
      challengeId: string;
      protocol: "dns" | "https" | "tls" | "tcp" | "udp";
      destinationRef: string;
      requestSummaryRef?: string;
      reasonCodes: string[];
      grantEligible: boolean;
      challengeExpiresAt: string;
    }
  | {
      eventId: string;
      type: "PolicyGrantReady";
      workspaceId: string;
      taskId: string;
      requestId: string;
      asyncId: string;
      challengeId: string;
      grantId: string;
      policyVersion: number;
      expiresAt: string;
      retryOriginalCommand: true;
    };

export interface CompletionCandidate {
  taskId: string;
  candidateVersion: number;
  ownerJudgement: string;
  outcomeRef: string;
  artifactRefs: string[];
  evidenceRefs: string[];
  contractVersion: number;
}

export interface AcceptanceReviewDecision {
  decision: "accept" | "reject" | "insufficient_evidence";
  rationale: string;
  unmetAcceptance: string[];
  requiredEvidence: string[];
  evidenceRefs: string[];
}

export interface CompletionReviewJob {
  reviewJobId: string;
  taskId: string;
  candidateVersion: number;
  candidateSnapshotRef: string;
  candidateDigest: string;
  inputSnapshotRef: string;
  inputDigest: string;
  contractVersion: number;
  reviewerProfileVersion: string;
  outputSchemaVersion: string;
  status: "pending" | "reviewing" | "completed" | "needs_operator";
  attempt: number;
  leaseOwner?: string;
  leaseExpiresAt?: string;
  invocationDeadlineAt: string;
  lastErrorRef?: string;
}

export interface CompletionReview {
  reviewId: string;
  reviewJobId: string;
  taskId: string;
  candidateVersion: number;
  reviewerProfileVersion: string;
  inputDigest: string;
  decision: AcceptanceReviewDecision;
  decidedAt: string;
}

export interface EgressAttempt {
  attemptId: string;
  workspaceId: string;
  taskId: string;
  agentId: string;
  originTaskId: string;
  delegationChainDigest: string;
  processRef: string;
  binding: EgressRequestBinding;
  dataClassification: string[];
  decision: "allow" | "block";
  policyRefs: string[];
  createdAt: string;
}

export interface EgressCaptureRange {
  start: number;
  endExclusive: number;
  digest: string;
}

export interface EgressCaptureCoverage {
  status: "complete" | "partial" | "unavailable" | "incomplete";
  totalBytes: number | null;
  capturedRanges: EgressCaptureRange[];
  redactedRanges: EgressCaptureRange[];
  truncated: boolean;
  limitationReason: string | null;
  classification: string[];
  encryptedBlobRef: string | null;
  keyRef: string | null;
}

export interface EgressCaptureManifest {
  captureManifestId: string;
  attemptId: string;
  request: EgressCaptureCoverage;
  response: EgressCaptureCoverage | null;
  completionStatus: "request_committed" | "response_committed" | "connection_failed" | "incomplete";
  retentionExpiresAt: string;
  pinnedUntilRef: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface EgressRuleDecision {
  ruleDecisionId: string;
  attemptId: string;
  decision: "allow" | "block";
  policyVersion: number;
  matchedRuleRefs: string[];
  reasonCodes: string[];
  evaluatedAt: string;
}

export interface PolicyRevisionDecision {
  decision: "update" | "no_change" | "require_authority";
  rationale: string;
  evidenceRefs: string[];
}

export interface EgressRequestBinding {
  protocol: "dns" | "https" | "tls" | "tcp" | "udp";
  scheme: string;
  fqdn: string | null;
  resolvedIp: string | null;
  port: number;
  method: string | null;
  normalizedPathQuery: string | null;
  policyHeadersDigest: string | null;
  bodyDigest: string | null;
  bodySize: number | null;
  requestedCredentialScope: string | null;
  dnsSnapshotRef: string | null;
  baselinePolicyVersion: number;
  canonicalRequestDigest: string;
}

export interface EgressChallenge {
  challengeId: string;
  workspaceId: string;
  taskId: string;
  originTaskId: string;
  delegationChainDigest: string;
  contractVersion: number;
  binding: EgressRequestBinding;
  destinationRef: string;
  requestSummaryRef?: string;
  reasonCodes: string[];
  grantEligible: boolean;
  autoGrantEligible: boolean;
  requiredAuthorityRef: string | null;
  createdAt: string;
  expiresAt: string;
}

export interface ChallengeObservation {
  challengeId: string;
  attemptId: string;
  observedAt: string;
}

export interface GrantRequest {
  requestId: string;
  workspaceId: string;
  taskId: string;
  originTaskId: string;
  delegationChainDigest: string;
  challengeId: string;
  asyncId: string;
  justification: string;
  evidenceRefs: string[];
  operationKey: string;
  status: "pending" | "evaluating" | "waiting_authority" | "completed" | "denied" | "cancelled";
  createdAt: string;
}

export interface GrantEvaluationJob {
  jobId: string;
  grantRequestId: string;
  inputSnapshotRef: string;
  inputDigest: string;
  profileVersion: string;
  outputSchemaVersion: string;
  status: "pending" | "evaluating" | "completed" | "cancelled" | "needs_operator";
  attempt: number;
  leaseOwner?: string;
  leaseExpiresAt?: string;
  invocationDeadlineAt: string;
  lastErrorRef?: string;
}

export interface GrantDecision {
  decision: "grant" | "deny" | "require_authority";
  rationale: string;
  question: string | null;
  evidenceRefs: string[];
}

export interface GrantDecisionRecord {
  decisionId: string;
  grantRequestId: string;
  evaluationJobId: string;
  inputDigest: string;
  profileVersion: string;
  outputSchemaVersion: string;
  decision: GrantDecision;
  decidedAt: string;
}

export interface GrantAuthorityRequest {
  authorityRequestId: string;
  grantRequestId: string;
  grantDecisionId: string;
  challengeId: string;
  bindingDigest: string;
  authorityRef: string;
  status: "pending" | "approved" | "denied" | "expired" | "cancelled";
  createdAt: string;
  expiresAt: string;
}

export interface GrantAuthorityDecision {
  authorityDecisionId: string;
  authorityRequestId: string;
  responderPrincipal: string;
  decision: "approve" | "deny";
  rationale: string;
  decidedAt: string;
}

export interface PolicyGrant {
  grantId: string;
  workspaceId: string;
  sourceTaskId: string;
  originTaskId: string;
  delegationChainDigest: string;
  contractVersion: number;
  sourceChallengeId: string;
  sourceGrantRequestId: string;
  sourceGrantDecisionId: string;
  sourceAuthorityDecisionId: string | null;
  bindingDigest: string;
  protocol: "dns" | "https" | "tls" | "tcp" | "udp";
  resolvedIp: string | null;
  port: number;
  credentialScope: string | null;
  maxUses: 1;
  connectionLimit: number;
  byteLimit: number | null;
  policyVersion: number;
  status: "pending_activation" | "active" | "revoked";
  useCount: number;
  createdAt: string;
  expiresAt: string;
  revokedAt?: string;
}

export interface EgressReviewJob {
  reviewJobId: string;
  watermark: string;
  selectionReason: "high_risk" | "anomaly" | "random_sample" | "incident_replay";
  inputSnapshotRef: string;
  inputDigest: string;
  profileVersion: string;
  outputSchemaVersion: string;
  status: "pending" | "reviewing" | "completed" | "needs_operator";
  attempt: number;
  leaseOwner: string | null;
  leaseExpiresAt: string | null;
  reviewDeadlineAt: string;
  capturePinRef: string;
  lastErrorRef: string | null;
  createdAt: string;
}

export interface EgressFinding {
  findingId: string;
  reviewJobId: string;
  attemptIds: string[];
  verdict: "benign" | "policy_bypass" | "suspicious" | "insufficient_evidence";
  severity: "low" | "medium" | "high" | "critical";
  rationale: string;
  evidenceRefs: string[];
  createdAt: string;
}

export interface PolicyRevision {
  revisionId: string;
  workspaceId: string | null;
  targetPolicyKey: string;
  sourceProposalId: string;
  sourceAuthorityDecisionId: string | null;
  basePolicyVersion: number;
  previousPolicyVersion: number;
  newPolicyVersion: number;
  targetPolicyRef: string;
  targetPolicyDigest: string;
  promptProfileVersion: string;
  regressionEvidenceRefs: string[];
  approvedBy: string;
  status: "pending_activation" | "active" | "superseded" | "cancelled";
  activationAckRef: string | null;
  createdAt: string;
  activatedAt: string | null;
}

export interface PolicyRevisionJob {
  jobId: string;
  workspaceId: string | null;
  targetPolicyKey: string;
  inputSnapshotRef: string;
  inputDigest: string;
  candidatePolicyRef: string | null;
  candidatePolicyDigest: string | null;
  basePolicyVersion: number | null;
  candidateFixedAt: string | null;
  profileVersion: string;
  outputSchemaVersion: string;
  status: "pending" | "reviewing" | "completed" | "cancelled" | "needs_operator";
  attempt: number;
  leaseOwner: string | null;
  leaseExpiresAt: string | null;
  invocationDeadlineAt: string;
  lastErrorRef: string | null;
}

export interface PolicyRevisionProposal {
  proposalId: string;
  jobId: string;
  workspaceId: string | null;
  targetPolicyKey: string;
  candidatePolicyRef: string | null;
  candidatePolicyDigest: string | null;
  basePolicyVersion: number;
  regressionEvidenceRefs: string[];
  decision: PolicyRevisionDecision;
  applicationStatus: "pending" | "waiting_authority" | "ready" | "stale" | "applied" | "denied" | "expired";
  createdAt: string;
}

export interface PolicyRevisionJobFinding {
  jobId: string;
  findingId: string;
}

export interface PolicyRevisionAuthorityRequest {
  authorityRequestId: string;
  proposalId: string;
  candidatePolicyDigest: string;
  workspaceId: string | null;
  targetPolicyKey: string;
  basePolicyVersion: number;
  authorityRef: string;
  status: "pending" | "approved" | "denied" | "expired" | "cancelled";
  createdAt: string;
  expiresAt: string;
}

export interface PolicyRevisionAuthorityDecision {
  authorityDecisionId: string;
  authorityRequestId: string;
  responderPrincipal: string;
  decision: "approve" | "deny";
  rationale: string;
  decidedAt: string;
}

export interface DnsResolution {
  resolutionId: string;
  workspaceId: string;
  taskId: string;
  fqdn: string;
  resolvedIps: string[];
  ttlSeconds: number;
  observedAt: string;
}

export interface OutboundTransaction {
  transactionId: string;
  attemptId: string;
  workspaceId: string;
  taskId: string;
  grantId?: string;
  requestBindingDigest: string;
  outerRequestDigest: string;
  responseDigest?: string;
  status: "intent_committed" | "forwarded" | "completed" | "failed" | "outcome_unknown";
  startedAt: string;
  completedAt?: string;
}

export interface WorkspaceSecurityPolicyBinding {
  workspaceId: string;
  profileRef: string;
  baselinePolicyVersion: number;
  pendingRevisionId: string | null;
  pendingPolicyVersion: number | null;
  status: "active" | "frozen" | "retired";
  version: number;
  updatedAt: string;
}

export interface GlobalSecurityPolicyBinding {
  targetPolicyKey: string;
  profileRef: string;
  activePolicyVersion: number;
  pendingRevisionId: string | null;
  pendingPolicyVersion: number | null;
  version: number;
  updatedAt: string;
}

export interface EpisodeStatement {
  text: string;
  sourceRefs: string[];
  epistemicStatus: "observed" | "owner_asserted" | "compiler_inferred";
}

export interface EvidenceRecord {
  evidenceId: string;
  kind:
    | "agent_run_item"
    | "tool_log"
    | "artifact"
    | "workspace_snapshot"
    | "decision"
    | "review"
    | "egress"
    | "episode";
  taskId?: string;
  contentType: string;
  contentDigest: string;
  byteLength: number;
  retentionClass: "long" | "policy" | "short";
  redactionStatus: "none" | "redacted" | "encrypted";
  createdAt: string;
}

export interface EvidenceBlobChunk {
  evidenceId: string;
  chunkIndex: number;
  content: Uint8Array;
}

export interface EvidenceQueryRequest {
  sql: string;
  params?: Array<string | number | boolean | null>;
  maxRows?: number;
}

export interface EvidenceQueryResult {
  columns: string[];
  rows: unknown[][];
  truncated: boolean;
  nextCursor?: string;
}

export interface TaskEpisode {
  episodeId: string;
  taskId: string;
  parentTaskId?: string;
  ownerAgentId: string;
  situation: {
    objective: string;
    acceptance: string;
    instructions?: string;
    initialContextSummary: string;
  };
  temporalContext: { startedAt: string; endedAt: string };
  course: {
    summary: string;
    importantTransitions: EpisodeTransition[];
    childEpisodeRefs: string[];
  };
  outcome: {
    status: "completed" | "cancelled";
    ownerJudgement: string;
    acceptanceReviewRef?: string;
    artifactRefs: string[];
    outboundTransactionRefs: string[];
  };
  surprises: EpisodeStatement[];
  decisions: EpisodeStatement[];
  unresolved: EpisodeStatement[];
  evidenceRefs: string[];
}

export interface EpisodeCompilationJob {
  jobId: string;
  taskId: string;
  status: "pending" | "investigating" | "validating" | "completed" | "needs_operator";
  stepCount: number;
  inputTokens: number;
  outputTokens: number;
  maxSteps: number;
  inputTokenBudget: number;
  outputTokenBudget: number;
  artifactReadByteBudget: number;
  deadlineAt: string;
  profileVersion: string;
  outputSchemaVersion: string;
  leaseOwner?: string;
  leaseExpiresAt?: string;
  evidenceRefs: string[];
  attempt: number;
  lastErrorRef?: string;
}

export interface EpisodeTransition {
  at: string;
  eventType: string;
  summary: string;
  sourceRefs: string[];
}
