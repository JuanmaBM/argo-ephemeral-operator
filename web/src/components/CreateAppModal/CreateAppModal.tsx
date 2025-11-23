import React, { useState } from 'react';
import {
  Modal,
  ModalVariant,
  Button,
  Form,
  FormGroup,
  TextInput,
  Alert,
  Tabs,
  Tab,
  TabTitleText,
  TextArea,
  Checkbox,
} from '@patternfly/react-core';
import { useCreateEnvironment } from '../../hooks/useEphemeralApps';

interface CreateAppModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const CreateAppModal: React.FC<CreateAppModalProps> = ({ isOpen, onClose }) => {
  const [activeTabKey, setActiveTabKey] = useState<string | number>(0);
  const [formData, setFormData] = useState({
    name: '',
    namespace: 'default',
    repoURL: '',
    path: '',
    targetRevision: 'main',
    expirationDate: '',
    namespaceName: '',
    // Secrets
    secrets: [] as Array<{ name: string; sourceNamespace?: string; values?: string }>,
    // ConfigMaps
    configMaps: [] as Array<{ name: string; sourceNamespace?: string; data?: string }>,
    // Sync Policy
    syncPolicyEnabled: true,
    prune: true,
    selfHeal: true,
  });

  const createMutation = useCreateEnvironment();

  const handleSubmit = async () => {
    try {
      // Parse secrets
      const secrets = formData.secrets
        .filter((s) => s.name)
        .map((s) => {
          const secret: any = { name: s.name };
          if (s.sourceNamespace) {
            secret.sourceNamespace = s.sourceNamespace;
          }
          if (s.values) {
            try {
              secret.values = JSON.parse(s.values);
            } catch (e) {
              // If not valid JSON, treat as single key-value
              const lines = s.values.split('\n');
              const valuesObj: Record<string, string> = {};
              lines.forEach((line) => {
                const [key, ...valueParts] = line.split(':');
                if (key && valueParts.length > 0) {
                  valuesObj[key.trim()] = valueParts.join(':').trim();
                }
              });
              secret.values = valuesObj;
            }
          }
          return secret;
        });

      // Parse configMaps
      const configMaps = formData.configMaps
        .filter((cm) => cm.name)
        .map((cm) => {
          const configMap: any = { name: cm.name };
          if (cm.sourceNamespace) {
            configMap.sourceNamespace = cm.sourceNamespace;
          }
          if (cm.data) {
            try {
              configMap.data = JSON.parse(cm.data);
            } catch (e) {
              // If not valid JSON, parse as key: value format
              const lines = cm.data.split('\n');
              const dataObj: Record<string, string> = {};
              lines.forEach((line) => {
                const [key, ...valueParts] = line.split(':');
                if (key && valueParts.length > 0) {
                  dataObj[key.trim()] = valueParts.join(':').trim();
                }
              });
              configMap.data = dataObj;
            }
          }
          return configMap;
        });

      await createMutation.mutateAsync({
        metadata: {
          name: formData.name,
          namespace: formData.namespace,
        },
        spec: {
          repoURL: formData.repoURL,
          path: formData.path,
          targetRevision: formData.targetRevision,
          expirationDate: formData.expirationDate,
          namespaceName: formData.namespaceName || undefined,
          secrets: secrets.length > 0 ? secrets : undefined,
          configMaps: configMaps.length > 0 ? configMaps : undefined,
          syncPolicy: formData.syncPolicyEnabled
            ? {
                automated: {
                  prune: formData.prune,
                  selfHeal: formData.selfHeal,
                },
              }
            : undefined,
        },
      });
      
      onClose();
      
      // Reset form
      setFormData({
        name: '',
        namespace: 'default',
        repoURL: '',
        path: '',
        targetRevision: 'main',
        expirationDate: '',
        namespaceName: '',
        secrets: [],
        configMaps: [],
        syncPolicyEnabled: true,
        prune: true,
        selfHeal: true,
      });
      setActiveTabKey(0);
    } catch (error) {
      console.error('Failed to create environment:', error);
    }
  };

  const isValid =
    formData.name && formData.repoURL && formData.path && formData.expirationDate;

  const addSecret = () => {
    setFormData({
      ...formData,
      secrets: [...formData.secrets, { name: '', sourceNamespace: '', values: '' }],
    });
  };

  const removeSecret = (index: number) => {
    setFormData({
      ...formData,
      secrets: formData.secrets.filter((_, i) => i !== index),
    });
  };

  const addConfigMap = () => {
    setFormData({
      ...formData,
      configMaps: [...formData.configMaps, { name: '', sourceNamespace: '', data: '' }],
    });
  };

  const removeConfigMap = (index: number) => {
    setFormData({
      ...formData,
      configMaps: formData.configMaps.filter((_, i) => i !== index),
    });
  };

  // Generate default expiration date (24 hours from now)
  const getDefaultExpiration = () => {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    return tomorrow.toISOString().slice(0, 16); // Format: YYYY-MM-DDTHH:mm
  };

  return (
    <Modal
      variant={ModalVariant.large}
      title="Create Ephemeral Environment"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button
          key="create"
          variant="primary"
          onClick={handleSubmit}
          isDisabled={!isValid || createMutation.isPending}
          isLoading={createMutation.isPending}
        >
          Create
        </Button>,
        <Button key="cancel" variant="link" onClick={onClose}>
          Cancel
        </Button>,
      ]}
    >
      {createMutation.isError && (
        <Alert variant="danger" title="Error creating environment" isInline>
          {(createMutation.error as Error).message}
        </Alert>
      )}

      <Tabs activeKey={activeTabKey} onSelect={(_, tabIndex) => setActiveTabKey(tabIndex)}>
        <Tab eventKey={0} title={<TabTitleText>Basic</TabTitleText>}>
          <Form style={{ marginTop: '1rem' }}>
            <FormGroup label="Name" isRequired fieldId="name">
              <TextInput
                isRequired
                type="text"
                id="name"
                value={formData.name}
                onChange={(_, value) => setFormData({ ...formData, name: value })}
                placeholder="my-feature-branch"
              />
            </FormGroup>

            <FormGroup label="Namespace" fieldId="namespace">
              <TextInput
                type="text"
                id="namespace"
                value={formData.namespace}
                onChange={(_, value) => setFormData({ ...formData, namespace: value })}
              />
            </FormGroup>

            <FormGroup label="Repository URL" isRequired fieldId="repoURL">
              <TextInput
                isRequired
                type="text"
                id="repoURL"
                placeholder="https://github.com/org/repo.git"
                value={formData.repoURL}
                onChange={(_, value) => setFormData({ ...formData, repoURL: value })}
              />
            </FormGroup>

            <FormGroup label="Path" isRequired fieldId="path">
              <TextInput
                isRequired
                type="text"
                id="path"
                placeholder="k8s/"
                value={formData.path}
                onChange={(_, value) => setFormData({ ...formData, path: value })}
              />
            </FormGroup>

            <FormGroup label="Target Revision" fieldId="targetRevision">
              <TextInput
                type="text"
                id="targetRevision"
                placeholder="main"
                value={formData.targetRevision}
                onChange={(_, value) => setFormData({ ...formData, targetRevision: value })}
              />
            </FormGroup>

            <FormGroup
              label="Expiration Date & Time"
              isRequired
              fieldId="expirationDate"
            >
              <div>
                <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                  When this environment should be automatically deleted (RFC3339 format)
                </p>
                <TextInput
                  isRequired
                  type="datetime-local"
                  id="expirationDate"
                  value={formData.expirationDate}
                  onChange={(_, value) => {
                    // Convert to RFC3339 format
                    const date = new Date(value);
                    const rfc3339 = date.toISOString();
                    setFormData({ ...formData, expirationDate: rfc3339 });
                  }}
                  placeholder={getDefaultExpiration()}
                />
                <small style={{ marginTop: '0.5rem', display: 'block' }}>
                  Current value (RFC3339): {formData.expirationDate || 'Not set'}
                </small>
              </div>
            </FormGroup>

            <FormGroup label="Namespace Name (optional)" fieldId="namespaceName">
              <div>
                <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                  Custom namespace name. If not provided, a random name will be generated.
                </p>
                <TextInput
                  type="text"
                  id="namespaceName"
                  placeholder="Leave empty for auto-generated"
                  value={formData.namespaceName}
                  onChange={(_, value) => setFormData({ ...formData, namespaceName: value })}
                />
              </div>
            </FormGroup>
          </Form>
        </Tab>

        <Tab eventKey={1} title={<TabTitleText>Secrets</TabTitleText>}>
          <div style={{ marginTop: '1rem' }}>
            <Button variant="secondary" onClick={addSecret} style={{ marginBottom: '1rem' }}>
              Add Secret
            </Button>

            {formData.secrets.map((secret, index) => (
              <div
                key={index}
                style={{
                  border: '1px solid #d2d2d2',
                  padding: '1rem',
                  marginBottom: '1rem',
                  borderRadius: '4px',
                }}
              >
                <FormGroup label="Secret Name" isRequired>
                  <TextInput
                    value={secret.name}
                    onChange={(_, value) => {
                      const newSecrets = [...formData.secrets];
                      newSecrets[index].name = value;
                      setFormData({ ...formData, secrets: newSecrets });
                    }}
                    placeholder="my-secret"
                  />
                </FormGroup>

                <FormGroup label="Source Namespace (optional)">
                  <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                    Copy from existing namespace. Leave empty if providing inline values.
                  </p>
                  <TextInput
                    value={secret.sourceNamespace}
                    onChange={(_, value) => {
                      const newSecrets = [...formData.secrets];
                      newSecrets[index].sourceNamespace = value;
                      setFormData({ ...formData, secrets: newSecrets });
                    }}
                    placeholder="shared-secrets"
                  />
                </FormGroup>

                <FormGroup label="Inline Values (optional)">
                  <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                    Format: key: value (one per line) or JSON
                  </p>
                  <TextArea
                    value={secret.values}
                    onChange={(_, value) => {
                      const newSecrets = [...formData.secrets];
                      newSecrets[index].values = value;
                      setFormData({ ...formData, secrets: newSecrets });
                    }}
                    rows={4}
                    placeholder="username: admin&#10;password: secret"
                  />
                </FormGroup>

                <Button variant="danger" onClick={() => removeSecret(index)}>
                  Remove
                </Button>
              </div>
            ))}
          </div>
        </Tab>

        <Tab eventKey={2} title={<TabTitleText>ConfigMaps</TabTitleText>}>
          <div style={{ marginTop: '1rem' }}>
            <Button variant="secondary" onClick={addConfigMap} style={{ marginBottom: '1rem' }}>
              Add ConfigMap
            </Button>

            {formData.configMaps.map((cm, index) => (
              <div
                key={index}
                style={{
                  border: '1px solid #d2d2d2',
                  padding: '1rem',
                  marginBottom: '1rem',
                  borderRadius: '4px',
                }}
              >
                <FormGroup label="ConfigMap Name" isRequired>
                  <TextInput
                    value={cm.name}
                    onChange={(_, value) => {
                      const newConfigMaps = [...formData.configMaps];
                      newConfigMaps[index].name = value;
                      setFormData({ ...formData, configMaps: newConfigMaps });
                    }}
                    placeholder="app-config"
                  />
                </FormGroup>

                <FormGroup label="Source Namespace (optional)">
                  <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                    Copy from existing namespace. Leave empty if providing inline data.
                  </p>
                  <TextInput
                    value={cm.sourceNamespace}
                    onChange={(_, value) => {
                      const newConfigMaps = [...formData.configMaps];
                      newConfigMaps[index].sourceNamespace = value;
                      setFormData({ ...formData, configMaps: newConfigMaps });
                    }}
                    placeholder="shared-configs"
                  />
                </FormGroup>

                <FormGroup label="Inline Data (optional)">
                  <p style={{ fontSize: '0.875rem', color: '#6a6e73', marginBottom: '0.5rem' }}>
                    Format: key: value (one per line) or JSON
                  </p>
                  <TextArea
                    value={cm.data}
                    onChange={(_, value) => {
                      const newConfigMaps = [...formData.configMaps];
                      newConfigMaps[index].data = value;
                      setFormData({ ...formData, configMaps: newConfigMaps });
                    }}
                    rows={6}
                    placeholder="DATABASE_HOST: postgres.svc&#10;LOG_LEVEL: debug"
                  />
                </FormGroup>

                <Button variant="danger" onClick={() => removeConfigMap(index)}>
                  Remove
                </Button>
              </div>
            ))}
          </div>
        </Tab>

        <Tab eventKey={3} title={<TabTitleText>Sync Policy</TabTitleText>}>
          <Form style={{ marginTop: '1rem' }}>
            <FormGroup fieldId="syncPolicyEnabled">
              <Checkbox
                label="Enable Automated Sync Policy"
                id="syncPolicyEnabled"
                isChecked={formData.syncPolicyEnabled}
                onChange={(_, checked) =>
                  setFormData({ ...formData, syncPolicyEnabled: checked })
                }
              />
            </FormGroup>

            {formData.syncPolicyEnabled && (
              <>
                <FormGroup fieldId="prune">
                  <Checkbox
                    label="Prune - Delete resources that are no longer defined in Git"
                    id="prune"
                    isChecked={formData.prune}
                    onChange={(_, checked) => setFormData({ ...formData, prune: checked })}
                  />
                </FormGroup>

                <FormGroup fieldId="selfHeal">
                  <Checkbox
                    label="Self Heal - Revert manual changes back to Git state"
                    id="selfHeal"
                    isChecked={formData.selfHeal}
                    onChange={(_, checked) => setFormData({ ...formData, selfHeal: checked })}
                  />
                </FormGroup>
              </>
            )}
          </Form>
        </Tab>
      </Tabs>
    </Modal>
  );
};
