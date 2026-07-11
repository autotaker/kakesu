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
  | "created"
  | "ready"
  | "running"
  | "waiting_child"
  | "waiting_parent"
  | "waiting_effect"
  | "suspended"
  | "reviewing_completion"
  | "waiting_evidence"
  | "cancellation_requested"
  | "completed"
  | "failed"
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
  startedAt: string;
  endedAt?: string;
}

export interface Continuation {
  continuationId: string;
  taskId: string;
  runId: string;
  reason:
    | "waiting_child"
    | "waiting_parent"
    | "waiting_effect"
    | "waiting_review"
    | "suspended";
  awaitedEventIds: string[];
  previousResponseId?: string;
  pendingCallId?: string;
  contractVersion: number;
  workspaceSnapshotRef: string;
  contextSnapshotRef: string;
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
  idempotencyKey?: string;
}

export interface CompletionCandidate {
  taskId: string;
  candidateVersion: number;
  ownerJudgement: string;
  outcomeRef: string;
  artifactRefs: string[];
  evidenceRefs: string[];
  contractVersion: number;
}

export type AcceptanceReviewDecision =
  | { decision: "accept"; rationale: string }
  | { decision: "reject"; rationale: string; unmetAcceptance: string[] }
  | {
      decision: "insufficient_evidence";
      rationale: string;
      requiredEvidence: string[];
    };

export interface NormalizedEffect {
  effectId: string;
  effectType: string;
  target: {
    provider: string;
    resourceType: string;
    resourceId: string;
  };
  operation: string;
  payloadRef: string;
  payloadDigest: string;
  payloadSummary: string;
  dataClassification: string[];
  estimatedCost?: number;
  requesterTaskId: string;
  originTaskId: string;
  delegationChain: string[];
  causalEventIds: string[];
  requesterExplanation?: string;
  evidenceRefs: string[];
}

export type PolicyDecision =
  | {
      decision: "allow";
      rationale: string;
      appliedPolicyIds: string[];
      conditions?: ExecutionCondition[];
    }
  | {
      decision: "deny";
      rationale: string;
      appliedPolicyIds: string[];
    }
  | {
      decision: "require_authority";
      rationale: string;
      appliedPolicyIds: string[];
      authorityRef: string;
      question: string;
    }
  | {
      decision: "insufficient_information";
      rationale: string;
      requiredEvidence: string[];
    };

export interface ExecutionCondition {
  type: string;
  value: string;
}

export interface EpisodeStatement {
  text: string;
  sourceRefs: string[];
  epistemicStatus: "observed" | "owner_asserted" | "compiler_inferred";
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
    status: "completed" | "failed" | "cancelled";
    ownerJudgement: string;
    acceptanceReviewRef?: string;
    artifactRefs: string[];
    effectRefs: string[];
  };
  surprises: EpisodeStatement[];
  decisions: EpisodeStatement[];
  unresolved: EpisodeStatement[];
  evidenceRefs: string[];
}

export interface EpisodeTransition {
  at: string;
  eventType: string;
  summary: string;
  sourceRefs: string[];
}
