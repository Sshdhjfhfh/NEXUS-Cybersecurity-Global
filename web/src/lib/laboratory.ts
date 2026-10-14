export interface LabStatus {
  ready: boolean;
  blockers: string[];
  roles: string[];
  classes: string[];
  passive_only: boolean;
}
export interface LabPrediction {
  class: string;
  probability: number;
  samples: number;
  lower: number;
  upper: number;
  ready: boolean;
}
export interface LabFinding {
  rule: string;
  severity: string;
  cwe: string;
  evidence_id: string;
  target: string;
}
export interface LabRun {
  id: string;
  task_id: number;
  target: string;
  evidence_ids: string[];
  evidence_digest: string;
  findings: LabFinding[];
  predictions: LabPrediction[];
  workers: { role: string; agent_id: string; state: string }[];
  started_at: string;
  finished_at: string;
}
