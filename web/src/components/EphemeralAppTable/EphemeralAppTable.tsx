import React, { useState } from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Button, Tooltip } from '@patternfly/react-core';
import { TrashIcon, ClockIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { formatDistanceToNow } from 'date-fns';
import type { EphemeralApplication } from '../../api/types';
import { StatusBadge } from '../StatusBadge/StatusBadge';
import { useDeleteEnvironment } from '../../hooks/useEphemeralApps';
import { UpdateAppModal } from '../CreateAppModal/UpdateAppModal';

interface EphemeralAppTableProps {
  environments: EphemeralApplication[];
}

export const EphemeralAppTable: React.FC<EphemeralAppTableProps> = ({ environments }) => {
  const navigate = useNavigate();
  const deleteMutation = useDeleteEnvironment();
  const [isUpdateModalOpen, setIsUpdateModalOpen] = useState(false);
  const [selectedApp, setSelectedApp] = useState<EphemeralApplication | null>(null);

  const handleDelete = async (name: string, namespace?: string) => {
    if (confirm(`Are you sure you want to delete ${name}?`)) {
      await deleteMutation.mutateAsync({ name, namespace });
    }
  };

  const handleUpdateExpiration = (env: EphemeralApplication) => {
    setSelectedApp(env);
    setIsUpdateModalOpen(true);
  };

  const handleCloseUpdateModal = () => {
    setIsUpdateModalOpen(false);
    setSelectedApp(null);
  };

  const getPhase = (env: EphemeralApplication) => {
    if (env.spec?.expirationDate && new Date(env.spec.expirationDate) < new Date()) {
      return 'Expiring';
    }
    return env.status?.phase;
  };

  return (
    <>
      <Table aria-label="Ephemeral Environments Table" variant="compact">
        <Thead>
          <Tr>
            <Th>Name</Th>
            <Th>Status</Th>
            <Th>Namespace</Th>
            <Th>Repository</Th>
            <Th>Target Revision</Th>
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
                <StatusBadge phase={getPhase(env)} />
              </Td>
              <Td dataLabel="Namespace">{env.status?.namespace || 'N/A'}</Td>
              <Td dataLabel="Repository">{env.spec.repoURL}</Td>
              <Td dataLabel="Target Revision">{env.spec.targetRevision}</Td>
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
                <Tooltip content="Update expiration date">
                  <Button
                    variant="plain"
                    icon={<ClockIcon />}
                    onClick={() => handleUpdateExpiration(env)}
                    style={{ marginRight: '0.5rem' }}
                  />
                </Tooltip>
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

      <UpdateAppModal
        isOpen={isUpdateModalOpen}
        onClose={handleCloseUpdateModal}
        ephemeralApp={selectedApp}
      />
    </>
  );
};

