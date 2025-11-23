import React from 'react';
import { Label } from '@patternfly/react-core';
import type { Phase } from '../../api/types';

interface StatusBadgeProps {
  phase?: Phase;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ phase = 'Pending' }) => {
  const getColor = (): 'green' | 'blue' | 'orange' | 'red' | 'grey' => {
    switch (phase) {
      case 'Active':
        return 'green';
      case 'Creating':
        return 'blue';
      case 'Expiring':
        return 'orange';
      case 'Failed':
        return 'red';
      default:
        return 'grey';
    }
  };

  return <Label color={getColor()}>{phase}</Label>;
};

