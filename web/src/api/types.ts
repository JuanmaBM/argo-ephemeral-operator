// TypeScript types matching the Go CRD

export interface EphemeralApplication {
  apiVersion: string;
  kind: string;
  metadata: Metadata;
  spec: EphemeralApplicationSpec;
  status: EphemeralApplicationStatus;
}

export interface Metadata {
  name: string;
  namespace?: string;
  creationTimestamp?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export interface EphemeralApplicationSpec {
  repoURL: string;
  path: string;
  targetRevision: string;
  expirationDate: string;
  namespaceName?: string;
  secrets?: SecretReference[];
  configMaps?: ConfigMapReference[];
  syncPolicy?: SyncPolicy;
}

export interface SyncPolicy {
  automated?: {
    prune?: boolean;
    selfHeal?: boolean;
  };
}

export interface SecretReference {
  name: string;
  sourceNamespace: string;
  targetName?: string;
  values?: Record<string, string>;
}

export interface ConfigMapReference {
  name: string;
  sourceNamespace?: string;
  data?: Record<string, string>;
}

export interface EphemeralApplicationStatus {
  phase?: Phase;
  namespace?: string;
  argoApplicationName?: string;
  message?: string;
  lastSyncTime?: string;
  conditions?: Condition[];
  copiedSecrets?: string[];
  copiedConfigMaps?: string[];
}

export type Phase = 'Pending' | 'Creating' | 'Active' | 'Expiring' | 'Failed';

export interface Condition {
  type: string;
  status: string;
  reason: string;
  message: string;
  lastTransitionTime: string;
}

export interface EphemeralApplicationList {
  apiVersion: string;
  kind: string;
  items: EphemeralApplication[];
}

export interface MetricsResponse {
  totalEnvironments: number;
  activeEnvironments: number;
  creatingEnvironments: number;
  failedEnvironments: number;
  environmentsByPhase: Record<string, number>;
  recentEnvironments: EnvironmentSummary[];
}

export interface EnvironmentSummary {
  name: string;
  namespace: string;
  phase: string;
  expirationDate: string;
  createdAt: string;
}

export interface CreateEnvironmentRequest {
  metadata: {
    name: string;
    namespace?: string;
  };
  spec: EphemeralApplicationSpec;
}

