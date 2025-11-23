import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ephemeralAppsApi } from '../api/ephemeralApps';
import type { CreateEnvironmentRequest, EphemeralApplication } from '../api/types';

const QUERY_KEY = 'ephemeralApps';

export const useEphemeralApps = () => {
  return useQuery({
    queryKey: [QUERY_KEY],
    queryFn: ephemeralAppsApi.list,
    refetchInterval: 30000, // Refetch every 30 seconds
  });
};

export const useEphemeralApp = (name: string, namespace = 'default') => {
  return useQuery({
    queryKey: [QUERY_KEY, name, namespace],
    queryFn: () => ephemeralAppsApi.get(name, namespace),
    enabled: !!name,
  });
};

export const useCreateEnvironment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateEnvironmentRequest) => ephemeralAppsApi.create(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
    },
  });
};

export const useDeleteEnvironment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, namespace = 'default' }: { name: string; namespace?: string }) =>
      ephemeralAppsApi.delete(name, namespace),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
    },
  });
};

export const useExtendExpiration = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      name,
      expirationDate,
      namespace = 'default',
    }: {
      name: string;
      expirationDate: string;
      namespace?: string;
    }) =>
      ephemeralAppsApi.update(
        name,
        {
          spec: { expirationDate } as EphemeralApplication['spec'],
        },
        namespace
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
    },
  });
};

export const useMetrics = () => {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: ephemeralAppsApi.getMetrics,
    refetchInterval: 60000, // Refetch every minute
  });
};

