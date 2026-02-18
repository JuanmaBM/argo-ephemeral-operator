import apiClient from './client';
import type {
  EphemeralApplicationList,
  EphemeralApplication,
  CreateEnvironmentRequest,
  MetricsResponse,
} from './types';

export const ephemeralAppsApi = {
  // List all ephemeral applications
  list: async (): Promise<EphemeralApplication[]> => {
    const { data } = await apiClient.get<EphemeralApplicationList>('/ephemeral-apps');
    return data.items || [];
  },

  // Get a single ephemeral application
  get: async (name: string, namespace = 'default'): Promise<EphemeralApplication> => {
    const { data } = await apiClient.get<EphemeralApplication>(
      `/ephemeral-apps/${name}?namespace=${namespace}`
    );
    return data;
  },

  // Create a new ephemeral application
  create: async (request: CreateEnvironmentRequest): Promise<EphemeralApplication> => {
    const { data } = await apiClient.post<EphemeralApplication>(
      '/ephemeral-apps/create',
      request
    );
    return data;
  },

  // Update an ephemeral application (extend expiration)
  update: async (
    name: string,
    updates: Partial<EphemeralApplication>,
    namespace = 'default'
  ): Promise<EphemeralApplication> => {
    const { data } = await apiClient.patch<EphemeralApplication>(
      `/ephemeral-apps/${name}?namespace=${namespace}`,
      updates
    );
    return data;
  },

  // Delete an ephemeral application
  delete: async (name: string, namespace = 'default'): Promise<void> => {
    await apiClient.delete(`/ephemeral-apps/${name}?namespace=${namespace}`);
  },

  // Get metrics
  getMetrics: async (): Promise<MetricsResponse> => {
    const { data } = await apiClient.get<MetricsResponse>('/metrics');
    return data;
  },
};

