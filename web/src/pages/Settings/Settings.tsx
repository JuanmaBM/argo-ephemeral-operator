import React, { useState, useEffect } from 'react';
import {
  PageSection,
  Title,
  Card,
  CardBody,
  Form,
  FormGroup,
  TextArea,
  Button,
  Alert,
  AlertActionCloseButton,
} from '@patternfly/react-core';

export const Settings: React.FC = () => {
  const [token, setToken] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);

  useEffect(() => {
    // Load existing token
    const savedToken = localStorage.getItem('k8s_token');
    if (savedToken) {
      setToken(savedToken);
    }
  }, []);

  const handleSave = () => {
    localStorage.setItem('k8s_token', token);
    setShowSuccess(true);
    setTimeout(() => setShowSuccess(false), 3000);
  };

  const handleClear = () => {
    setToken('');
    localStorage.removeItem('k8s_token');
    setShowSuccess(false);
  };

  return (
    <>
      <PageSection variant="light">
        <Title headingLevel="h1" size="2xl">
          Authentication
        </Title>
      </PageSection>

      <PageSection>
        <Card>
          <CardBody>
            {showSuccess && (
              <Alert
                variant="success"
                title="Token saved successfully"
                actionClose={<AlertActionCloseButton onClose={() => setShowSuccess(false)} />}
                style={{ marginBottom: '1rem' }}
              />
            )}

            <Title headingLevel="h2" size="lg" style={{ marginBottom: '1rem' }}>
              Kubernetes Authentication
            </Title>

            <p style={{ marginBottom: '1rem' }}>
              Enter your Kubernetes ServiceAccount token to authenticate with the API.
            </p>

            <Alert
              variant="info"
              title="How to get a token"
              isInline
              style={{ marginBottom: '1rem' }}
            >
              <p>Run these commands to create a ServiceAccount and get a token:</p>
              <pre style={{ marginTop: '0.5rem', background: '#f5f5f5', padding: '1rem' }}>
                {`kubectl create serviceaccount ephemeral-user -n default
kubectl create clusterrolebinding ephemeral-user-binding \\
  --clusterrole=cluster-admin \\
  --serviceaccount=default:ephemeral-user
kubectl create token ephemeral-user -n default --duration=24h`}
              </pre>
            </Alert>

            <Form>
              <FormGroup label="ServiceAccount Token" isRequired fieldId="token">
                <TextArea
                  isRequired
                  type="text"
                  id="token"
                  name="token"
                  value={token}
                  onChange={(_, value) => setToken(value)}
                  rows={8}
                  placeholder="Paste your Kubernetes token here..."
                />
              </FormGroup>

              <div style={{ display: 'flex', gap: '1rem', marginTop: '1rem' }}>
                <Button variant="primary" onClick={handleSave} isDisabled={!token}>
                  Save Token
                </Button>
                <Button variant="secondary" onClick={handleClear}>
                  Clear Token
                </Button>
              </div>
            </Form>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

