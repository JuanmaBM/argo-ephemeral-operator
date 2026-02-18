import React, { useState } from 'react';
import {
  PageSection,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  Button,
  Card,
  CardBody,
  EmptyState,
  EmptyStateIcon,
  EmptyStateBody,
  Spinner,
  Alert,
} from '@patternfly/react-core';
import { PlusCircleIcon, CubesIcon } from '@patternfly/react-icons';
import { useEphemeralApps, useMetrics } from '../../hooks/useEphemeralApps';
import { EphemeralAppTable } from '../../components/EphemeralAppTable/EphemeralAppTable';
import { CreateAppModal } from '../../components/CreateAppModal/CreateAppModal';
import { MetricsCards } from '../../components/MetricsCards/MetricsCards';

export const Dashboard: React.FC = () => {
  const { data: environments, isLoading, error } = useEphemeralApps();
  const { data: metrics } = useMetrics();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  if (isLoading) {
    return (
      <PageSection isFilled>
        <Spinner size="xl" />
      </PageSection>
    );
  }

  if (error) {
    return (
      <PageSection>
        <Alert variant="danger" title="Error loading environments">
          {(error as Error).message}
        </Alert>
      </PageSection>
    );
  }

  return (
    <>
      <PageSection variant="light">
        <Title headingLevel="h1" size="2xl">
          Ephemeral Environments
        </Title>
      </PageSection>

      {metrics && (
        <PageSection>
          <MetricsCards metrics={metrics} />
        </PageSection>
      )}

      <PageSection>
        <Card>
          <CardBody>
            <Toolbar>
              <ToolbarContent>
                <ToolbarItem>
                  <Button
                    variant="primary"
                    icon={<PlusCircleIcon />}
                    onClick={() => setIsCreateModalOpen(true)}
                  >
                    Create Environment
                  </Button>
                </ToolbarItem>
              </ToolbarContent>
            </Toolbar>

            {environments && environments.length > 0 ? (
              <EphemeralAppTable environments={environments} />
            ) : (
              <EmptyState>
                <EmptyStateIcon icon={CubesIcon} />
                <Title headingLevel="h2" size="lg">
                  No ephemeral environments
                </Title>
                <EmptyStateBody>
                  Create your first ephemeral environment to get started.
                </EmptyStateBody>
                <Button variant="primary" onClick={() => setIsCreateModalOpen(true)}>
                  Create Environment
                </Button>
              </EmptyState>
            )}
          </CardBody>
        </Card>
      </PageSection>

      <CreateAppModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
      />
    </>
  );
};

