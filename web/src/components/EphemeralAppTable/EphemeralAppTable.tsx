import React from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Button, Tooltip } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { formatDistanceToNow } from 'date-fns';
import type { EphemeralApplication } from '../../api/types';
import { StatusBadge } from '../StatusBadge/StatusBadge';
import { useDeleteEnvironment } from '../../hooks/useEphemeralApps';

interface EphemeralAppTableProps {
  environments: EphemeralApplication[];
}

export const EphemeralAppTable: React.FC<EphemeralAppTableProps> = ({ environments }) => {
  const navigate = useNavigate();
  const deleteMutation = useDeleteEnvironment();

  const handleDelete = async (name: string, namespace?: string) => {
    if (confirm(`Are you sure you want to delete ${name}?`)) {
      await deleteMutation.mutateAsync({ name, namespace });
    }
  };

  return (
    <Table aria-label="Ephemeral Environments Table" variant="compact">
      <Thead>
        <Tr>
          <Th>Name</Th>
          <Th>Status</Th>
          <Th>Namespace</Th>
          <Th>Repository</Th>
          <Th>Expires</Th>
          <Th>Created</Th>
          <Th>Actions</Th>
        </Tr>
      </Thead>
      <Tbody>
        {environments.map((env) => (
          <Tr key={env.metadata.name}>
            <Td
              dataLabel="Name"
              onClick={() => navigate(`/environments/${env.metadata.name}`)}
              style={{ cursor: 'pointer', color: '#06c' }}
            >
              {env.metadata.name}
            </Td>
            <Td dataLabel="Status">
              <StatusBadge phase={env.status?.phase} />
            </Td>
            <Td dataLabel="Namespace">{env.status?.namespace || 'N/A'}</Td>
            <Td dataLabel="Repository">{env.spec.repoURL}</Td>
            <Td dataLabel="Expires">
              {env.spec.expirationDate
                ? formatDistanceToNow(new Date(env.spec.expirationDate), {
                    addSuffix: true,
                  })
                : 'N/A'}
            </Td>
            <Td dataLabel="Created">
              {env.metadata.creationTimestamp
                ? formatDistanceToNow(new Date(env.metadata.creationTimestamp), {
                    addSuffix: true,
                  })
                : 'N/A'}
            </Td>
            <Td dataLabel="Actions">
              <Tooltip content="Delete environment">
                <Button
                  variant="plain"
                  icon={<TrashIcon />}
                  onClick={() => handleDelete(env.metadata.name, env.metadata.namespace)}
                  isDisabled={deleteMutation.isPending}
                />
              </Tooltip>
            </Td>
          </Tr>
        ))}
      </Tbody>
    </Table>
  );
};

