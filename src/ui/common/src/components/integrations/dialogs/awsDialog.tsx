import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React, { useEffect, useState } from 'react';

import {
  AWSConfig,
  AWSCredentialType,
  DynamicEngineType,
  DynamicK8sConfig,
} from '../../../utils/integrations';
import { Tab, Tabs } from '../../primitives/Tabs.styles';
import { IntegrationTextInputField } from './IntegrationTextInputField';

const Placeholders: AWSConfig = {
  type: AWSCredentialType.AccessKey,
  region: 'us-east-2',
  access_key_id: '',
  secret_access_key: '',
  config_file_path: '',
  config_file_profile: '',
  k8s_serialized: '',
};

const K8sPlaceholders: DynamicK8sConfig = {
  keepalive: '1200',
  cpu_node_type: 't3.xlarge',
  gpu_node_type: 'p2.xlarge',
  min_cpu_node: '1',
  max_cpu_node: '1',
  min_gpu_node: '0',
  max_gpu_node: '1',
};

type Props = {
  onUpdateField: (field: keyof AWSConfig, value: string) => void;
  value?: AWSConfig;
};

export const AWSDialog: React.FC<Props> = ({ onUpdateField, value }) => {
  const [engineType, setEngineType] = useState(DynamicEngineType.K8s);

  useEffect(() => {
    if (!value?.type) {
      onUpdateField('type', AWSCredentialType.AccessKey);
    }
  }, [onUpdateField, value?.type]);

  const k8sConfigs = JSON.parse(value?.k8s_serialized ?? '{}') as {
    [key: string]: string;
  };

  const configProfileInput = (
    <IntegrationTextInputField
      spellCheck={false}
      required={true}
      label="AWS Profile*"
      description="The name of the profile specified in brackets in your credential file."
      placeholder={Placeholders.config_file_profile}
      onChange={(event) =>
        onUpdateField('config_file_profile', event.target.value)
      }
      value={value?.config_file_profile ?? ''}
    />
  );

  const accessKeyTab = (
    <Box>
      <Typography variant="body2" color="gray.700">
        Manually enter your AWS credentials.
      </Typography>
      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="AWS Access Key ID*"
        description="The access key ID of your AWS account."
        placeholder={Placeholders.access_key_id}
        onChange={(event) => onUpdateField('access_key_id', event.target.value)}
        value={value?.access_key_id ?? ''}
      />

      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="AWS Secret Access Key*"
        description="The secret access key of your AWS account."
        placeholder={Placeholders.secret_access_key}
        onChange={(event) =>
          onUpdateField('secret_access_key', event.target.value)
        }
        value={value?.secret_access_key ?? ''}
      />

      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="AWS Region*"
        description="The region of your AWS account."
        placeholder={Placeholders.region}
        onChange={(event) => onUpdateField('region', event.target.value)}
        value={value?.region ?? ''}
      />
    </Box>
  );

  const configPathTab = (
    <Box>
      <Typography variant="body2" color="gray.700">
        Specify the path to your AWS credentials <strong>on the machine</strong>{' '}
        where you are running the Aqueduct server. Typically, this is in{' '}
        <code>~/.aws/credentials</code>, or <code>~/.aws/config</code> for SSO.
        You also need to specify the profile name you would like to use for the
        credentials file. Once connected, any updates to the file content will
        automatically apply to this integration.
      </Typography>
      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="AWS Credentials File Path*"
        description={'The path to the credentials file'}
        placeholder={Placeholders.config_file_path}
        onChange={(event) =>
          onUpdateField('config_file_path', event.target.value)
        }
        value={value?.config_file_path ?? ''}
      />

      {configProfileInput}
    </Box>
  );

  const k8sConfigTab = (
    <Box>
      <Typography variant="body2" color="gray.700">
        Optionally configure on-demand Kubernetes cluster parameters.
      </Typography>
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="Keepalive period"
        description="How long (in seconds) does the cluster need to remain idle before it is deleted."
        placeholder={K8sPlaceholders.keepalive}
        onChange={(event) => {
          k8sConfigs['keepalive'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['keepalive'] ?? ''}
      />
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="CPU node type"
        description="The EC2 instance type of the CPU node group."
        placeholder={K8sPlaceholders.cpu_node_type}
        onChange={(event) => {
          k8sConfigs['cpu_node_type'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['cpu_node_type'] ?? ''}
      />

      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="GPU node type"
        description="The EC2 instance type of the GPU node group."
        placeholder={K8sPlaceholders.gpu_node_type}
        onChange={(event) => {
          k8sConfigs['gpu_node_type'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['gpu_node_type'] ?? ''}
      />
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="Min CPU node"
        description="Minimum number of nodes in the CPU node group."
        placeholder={K8sPlaceholders.min_cpu_node}
        onChange={(event) => {
          k8sConfigs['min_cpu_node'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['min_cpu_node'] ?? ''}
      />
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="Max CPU node"
        description="Maximum number of nodes in the CPU node group."
        placeholder={K8sPlaceholders.max_cpu_node}
        onChange={(event) => {
          k8sConfigs['max_cpu_node'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['max_cpu_node'] ?? ''}
      />
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="Min GPU node"
        description="Minimum number of nodes in the GPU node group."
        placeholder={K8sPlaceholders.min_gpu_node}
        onChange={(event) => {
          k8sConfigs['min_gpu_node'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['min_gpu_node'] ?? ''}
      />
      <IntegrationTextInputField
        spellCheck={false}
        required={false}
        label="Max GPU node"
        description="Maximum number of nodes in the GPU node group."
        placeholder={K8sPlaceholders.max_gpu_node}
        onChange={(event) => {
          k8sConfigs['max_gpu_node'] = event.target.value;
          onUpdateField('k8s_serialized', JSON.stringify(k8sConfigs));
        }}
        value={k8sConfigs['max_gpu_node'] ?? ''}
      />
    </Box>
  );

  return (
    <Box sx={{ mt: 2 }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs
          value={value?.type ?? 'access_key'}
          onChange={(_, value) => onUpdateField('type', value)}
        >
          <Tab value={AWSCredentialType.AccessKey} label="Enter Access Keys" />
          <Tab
            value={AWSCredentialType.ConfigFilePath}
            label="Specify Path to Credentials"
          />
        </Tabs>
      </Box>
      {value?.type === AWSCredentialType.AccessKey && accessKeyTab}
      {value?.type === AWSCredentialType.ConfigFilePath && configPathTab}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs value={engineType} onChange={(_, value) => setEngineType(value)}>
          <Tab
            value={DynamicEngineType.K8s}
            label="On-demand Kubernetes Cluster Config"
          />
        </Tabs>
      </Box>
      {engineType === DynamicEngineType.K8s && k8sConfigTab}
    </Box>
  );
};

export function isAWSConfigComplete(config: AWSConfig): boolean {
  if (config.type === AWSCredentialType.AccessKey) {
    return (
      !!config.access_key_id && !!config.secret_access_key && !!config.region
    );
  }

  if (config.type === AWSCredentialType.ConfigFilePath) {
    return !!config.config_file_profile && !!config.config_file_path;
  }

  return false;
}
