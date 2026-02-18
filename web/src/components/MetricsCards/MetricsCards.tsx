import React from 'react';
import { Gallery, Card, CardBody, CardTitle } from '@patternfly/react-core';
import type { MetricsResponse } from '../../api/types';

interface MetricsCardsProps {
  metrics: MetricsResponse;
}

export const MetricsCards: React.FC<MetricsCardsProps> = ({ metrics }) => {
  return (
    <Gallery hasGutter minWidths={{ default: '200px' }}>
      <Card>
        <CardTitle>Total Environments</CardTitle>
        <CardBody>
          <div style={{ fontSize: '2rem', fontWeight: 'bold' }}>
            {metrics.totalEnvironments}
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardTitle>Active</CardTitle>
        <CardBody>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#3E8635' }}>
            {metrics.activeEnvironments}
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardTitle>Creating</CardTitle>
        <CardBody>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#2B9AF3' }}>
            {metrics.creatingEnvironments}
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardTitle>Failed</CardTitle>
        <CardBody>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#C9190B' }}>
            {metrics.failedEnvironments}
          </div>
        </CardBody>
      </Card>
    </Gallery>
  );
};

