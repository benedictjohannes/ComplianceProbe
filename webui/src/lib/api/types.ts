/**
 * API and Domain Types strictly mirroring the Go backend DTOs, schemas, and SSE payloads.
 */

// Lifecycle Statuses
export type Status =
  | 'idle'
  | 'loaded'
  | 'running.elevating'
  | 'running'
  | 'running.cancelling'
  | 'completed.confirming_submission'
  | 'completed.submitting'
  | 'completed.submitted'
  | 'completed.submission_error'
  | 'completed'
  | 'error';

// Structured Error Codes
export type ErrorCode =
  | 'PLAYBOOK_PARSE_FAILED'
  | 'PLAYBOOK_VALIDATION_FAILED'
  | 'REMOTE_FETCH_FAILED'
  | 'ELEVATION_DENIED'
  | 'ELEVATION_TIMEOUT'
  | 'ELEVATION_FAILED'
  | 'EXECUTION_ABORTED'
  | 'EXECUTION_FAILED'
  | 'FOLDER_WRITE_FAILED'
  | 'REMOTE_SUBMISSION_FAILED'
  | 'REMOTE_SUBMISSION_TIMEOUT'
  | 'CONFLICT'
  | 'INTERNAL_ERROR'
  | 'UNAUTHORIZED'
  | 'NO_REPORT';

export interface ValidationError {
  path: string;
  code: string;
  message: string;
}

export interface AppError {
  code: ErrorCode | string;
  message: string;
  detail?: ValidationError[] | unknown;
}

// Report Destinations
export type FolderSource = 'default' | 'cli' | 'playbook' | 'custom' | 'off';
export type HttpsSource = 'playbook' | 'custom' | 'off';
export type ReportFormat = 'multipart' | 'json';

export interface HttpsDestinationConfig {
  url: string;
  format?: ReportFormat | 'json' | 'multipart';
  secret?: string;
  headers?: Record<string, string>;
}

export interface PlaybookHttpsInspection {
  url: string;
  format: string; // "json" | "multipart"
  hasSignatureSecret: boolean;
  configuredHeaders: string[];
}

export interface PlaybookDestinationDefaults {
  has_folder: boolean;
  folder_path?: string;
  has_https: boolean;
  https?: PlaybookHttpsInspection;
}

export interface ReportDestinationState {
  folder?: string;
  folder_source: FolderSource;
  https_source: HttpsSource;
  https?: HttpsDestinationConfig;
  playbook_defaults?: PlaybookDestinationDefaults;
}

export interface DestinationUpdateRequest {
  folder?: string;
  folder_source?: FolderSource;
  https_source?: HttpsSource;
  https?: HttpsDestinationConfig;
}

export interface RemotePlaybookRequest {
  url: string;
  headers?: Record<string, string>;
}

export interface AppStateResponse {
  status: Status;
  errors: AppError[];
  report_destination: ReportDestinationState;
  active_run_id?: string;
}

// Playbook Structure
export interface EvaluationRule {
  regex?: string;
  includeStdErr?: boolean;
  func?: string;
  funcFile?: string;
}

export interface GatherSpec {
  key: string;
  excludeFromReport?: boolean;
  regex?: string;
  includeStdErr?: boolean;
  func?: string;
  funcFile?: string;
}

export interface ExitCodeRule {
  min?: number;
  max?: number;
  result: number;
}

export interface Exec {
  shell?: string;
  shellFunc?: string;
  shellFuncFile?: string;
  script?: string;
  scriptFileExtension?: string;
  func?: string;
  funcFile?: string;
  gather?: GatherSpec[];
  excludeFromReport?: boolean;
  requireElevation?: boolean;
}

export interface Cmd {
  exec: Exec;
  passScore?: number;
  failScore?: number;
  stdOutRule?: EvaluationRule;
  stdErrRule?: EvaluationRule;
  exitCodeRules?: ExitCodeRule[];
}

export interface Assertion {
  code: string;
  title: string;
  description: string;
  preCmds?: Exec[];
  cmds: Cmd[];
  postCmds?: Exec[];
  minPassingScore?: number;
  passDescription: string;
  failDescription: string;
}

export interface Section {
  title: string;
  description: string[];
  assertions: Assertion[];
}

export type ReportDestination = 'folder' | 'https';

export interface ReportDestinationConfig {
  url: string;
  format?: ReportFormat;
  signatureSecret?: string;
  additionalHeaders?: Record<string, string>;
}

export interface Playbook {
  title: string;
  reportFrontmatter?: Record<string, unknown>;
  sections: Section[];
  reportDestination?: ReportDestination;
  reportDestinationFolder?: string;
  reportDestinationHttps?: ReportDestinationConfig;
}

export interface PlaybookInspection {
  title: string;
  reportFrontmatter?: Record<string, unknown>;
  sections: Section[];
  reportDestination?: ReportDestination;
  reportDestinationFolder?: string;
  reportDestinationHttps?: PlaybookHttpsInspection;
  requiresElevation: boolean;
}

// Execution & Progression
export type AssertionExecutionStatus = 'pending' | 'running' | 'passed' | 'failed' | 'cancelled';

export interface AssertionSnapshot {
  code: string;
  title: string;
  status: AssertionExecutionStatus;
  passed: boolean;
  score: number;
  min_score: number;
  duration_ms: number;
  output?: string;
}

export interface ExecutionSnapshot {
  run_id: string;
  status: string;
  active_assertion_code?: string;
  last_event_id: number;
  duration_ms: number;
  assertions: AssertionSnapshot[];
  logs: string[];
}

// SSE Event Stream Types
export type SSEEventType =
  | 'state_change'
  | 'assertion_progress'
  | 'log'
  | 'execution_completed'
  | 'execution_cancelled'
  | 'termination';

export interface SSEEvent<T = unknown> {
  id: number;
  type: SSEEventType;
  run_id?: string;
  data: T;
}

export interface AssertionProgressEventData {
  run_id: string;
  code: string;
  status: AssertionExecutionStatus;
  passed: boolean;
  score: number;
  min_score: number;
  duration_ms: number;
  output?: string;
}

export interface LogEventData {
  run_id: string;
  message: string;
}

export interface ExecutionCompletedEventData {
  run_id: string;
  status: string;
  duration_ms: number;
}

export interface ExecutionCancelledEventData {
  run_id: string;
}

export interface TerminationEventData {
  reason?: string;
}

// Report Data Types
export interface ReportTimestamps {
  start: string;
  end: string;
}

export interface ReportAssertion {
  timestamps: ReportTimestamps;
  passed: boolean;
  score: number;
  minScore: number;
  context: Record<string, unknown>;
}

export interface ReportStats {
  passed: number;
  failed: number;
}

export interface FinalReport {
  timestamps: ReportTimestamps;
  username: string;
  os: string;
  arch: string;
  assertions: Record<string, ReportAssertion>;
  stats: ReportStats;
}

export type ArchiveFormat = 'zip' | 'tar.gz' | 'tar.zst' | 'tar';

export interface ShutdownResponse {
  status: string;
}

export type ConnectionStatus = 'connecting' | 'connected' | 'reconnecting' | 'disconnected';
