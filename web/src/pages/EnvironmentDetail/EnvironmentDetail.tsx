import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  PageSection,
  Title,
  Breadcrumb,
  BreadcrumbItem,
  Card,
  CardBody,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Spinner,
  Alert,
  Button,
  Flex,
  FlexItem,
} from '@patternfly/react-core';
import { ArrowLeftIcon } from '@patternfly/react-icons';
import { useEphemeralApp } from '../../hooks/useEphemeralApps';
import { StatusBadge } from '../../components/StatusBadge/StatusBadge';
import { formatDistanceToNow } from 'date-fns';

export const EnvironmentDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const { data: environment, isLoading, error } = useEphemeralApp(name || '');

  if (isLoading) {
    return (
      <PageSection isFilled>
        <Spinner size="xl" />
      </PageSection>
    );
  }

  if (error || !environment) {
    return (
      <PageSection>
        <Alert variant="danger" title="Error loading environment">
          {(error as Error)?.message || 'Environment not found'}
        </Alert>
      </PageSection>
    );
  }

  return (
    <>
      <PageSection variant="light">
        <Flex>
          <FlexItem>
            <Button
              variant="plain"
              icon={<ArrowLeftIcon />}
              onClick={() => navigate('/environments')}
            >
              Back
            </Button>
          </FlexItem>
        </Flex>
        <Breadcrumb>
          <BreadcrumbItem to="/environments">Environments</BreadcrumbItem>
          <BreadcrumbItem isActive>{name}</BreadcrumbItem>
        </Breadcrumb>
        <Title headingLevel="h1" size="2xl">
          {name}
        </Title>
      </PageSection>

      <PageSection>
        <Card>
          <CardBody>
            <DescriptionList isHorizontal>
              <DescriptionListGroup>
                <DescriptionListTerm>Status</DescriptionListTerm>
                <DescriptionListDescription>
                  <StatusBadge phase={environment.status?.phase} />
                </DescriptionListDescription>
              </DescriptionListGroup>

              <DescriptionListGroup>
                <DescriptionListTerm>Namespace</DescriptionListTerm>
                <DescriptionListDescription>
                  {environment.status?.namespace || 'N/A'}
                </DescriptionListDescription>
              </DescriptionListGroup>

              <DescriptionListGroup>
                <DescriptionListTerm>Repository</DescriptionListTerm>
                <DescriptionListDescription>
                  {environment.spec.repoURL}
                </DescriptionListDescription>
              </DescriptionListGroup>

              <DescriptionListGroup>
                <DescriptionListTerm>Path</DescriptionListTerm>
                <DescriptionListDescription>
                  {environment.spec.path}
                </DescriptionListDescription>
              </DescriptionListGroup>

              <DescriptionListGroup>
                <DescriptionListTerm>Revision</DescriptionListTerm>
                <DescriptionListDescription>
                  {environment.spec.targetRevision}
                </DescriptionListDescription>
              </DescriptionListGroup>

              <DescriptionListGroup>
                <DescriptionListTerm>Expires</DescriptionListTerm>
                <DescriptionListDescription>
                  {environment.spec.expirationDate
                    ? formatDistanceToNow(new Date(environment.spec.expirationDate), {
                        addSuffix: true,
                      })
                    : 'N/A'}
                </DescriptionListDescription>
              </DescriptionListGroup>

              {environment.status?.message && (
                <DescriptionListGroup>
                  <DescriptionListTerm>Message</DescriptionListTerm>
                  <DescriptionListDescription>
                    {environment.status.message}
                  </DescriptionListDescription>
                </DescriptionListGroup>
              )}
            </DescriptionList>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

