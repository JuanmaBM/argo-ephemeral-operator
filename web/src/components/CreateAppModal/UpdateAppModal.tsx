import React, { useState, useEffect } from 'react';
import {
  Modal,
  ModalVariant,
  Button,
  Form,
  FormGroup,
  TextInput,
  Alert,
} from '@patternfly/react-core';
import { useExtendExpiration } from '../../hooks/useEphemeralApps';
import { EphemeralApplication } from '@/api/types';

interface UpdateAppModalProps {
  isOpen: boolean;
  onClose: () => void;
  ephemeralApp: EphemeralApplication | null;
}

export const UpdateAppModal: React.FC<UpdateAppModalProps> = ({ 
  isOpen, 
  onClose, 
  ephemeralApp 
}) => {
  const [expirationDateTime, setExpirationDateTime] = useState('');

  const extendMutation = useExtendExpiration();

  // Update the value when the app changes
  useEffect(() => {
    if (ephemeralApp?.spec.expirationDate) {
      // Convert from RFC3339 (UTC) to datetime-local format (local time)
      const date = new Date(ephemeralApp.spec.expirationDate);
      // Adjust for timezone offset to get local time
      const offset = date.getTimezoneOffset() * 60000; // offset in milliseconds
      const localDate = new Date(date.getTime() - offset);
      const formattedDate = localDate.toISOString().slice(0, 16);
      setExpirationDateTime(formattedDate);
    }
  }, [ephemeralApp]);

  const handleSubmit = async () => {
    if (!ephemeralApp) return;

    try {
      // Convert from datetime-local (local time) to RFC3339 (UTC)
      // The input datetime-local gives us a string like "2024-11-25T15:00"
      // which is already in local time, so we just create a Date object
      const date = new Date(expirationDateTime);
      const expirationDateRFC3339 = date.toISOString();

      await extendMutation.mutateAsync({
        name: ephemeralApp.metadata.name,
        namespace: ephemeralApp.metadata.namespace || 'default',
        expirationDate: expirationDateRFC3339,
      });

      onClose();
    } catch (error) {
      console.error('Failed to update environment:', error);
    }
  };


  return (
    <Modal
      variant={ModalVariant.medium}
      title="Update Expiration Date"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button
          key="update"
          variant="primary"
          onClick={handleSubmit}
          isDisabled={!expirationDateTime || extendMutation.isPending}
          isLoading={extendMutation.isPending}
        >
          Update
        </Button>,
        <Button key="cancel" variant="link" onClick={onClose}>
          Cancel
        </Button>,
      ]}
    >
      {extendMutation.isError && (
        <Alert variant="danger" title="Error updating environment" isInline>
          {(extendMutation.error as Error).message}
        </Alert>
      )}

      <Form style={{ marginTop: '1rem' }}>
        <FormGroup
          label="New Expiration Date & Time"
          isRequired
          fieldId="expirationDateTime"
        >
          <div>
            <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
              When this environment should be automatically deleted
            </p>
            <TextInput
              isRequired
              type="datetime-local"
              id="expirationDateTime"
              value={expirationDateTime}
              onChange={(_event, value) => setExpirationDateTime(value)}
            />
          </div>
        </FormGroup>
      </Form>
    </Modal>
  );
};
