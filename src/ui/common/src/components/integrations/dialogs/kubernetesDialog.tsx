import Box from '@mui/material/Box';
import React, { useEffect, useState } from 'react';

import {
  IntegrationConfig,
  KubernetesConfig,
} from '../../../utils/integrations';
import { IntegrationTextInputField } from './IntegrationTextInputField';

const Placeholders: KubernetesConfig = {
  kubeconfig_path: 'home/ubuntu/.kube/config',
  cluster_name: 'aqueduct',
};

type Props = {
  setDialogConfig: (config: IntegrationConfig) => void;
};

export const KubernetesDialog: React.FC<Props> = ({ setDialogConfig }) => {
  const [kubeconfig_path, setKubeConfigPath] = useState<string>(null);
  const [cluster_name, setClusterName] = useState<string>(null);

  useEffect(() => {
    const config: KubernetesConfig = {
      kubeconfig_path: kubeconfig_path,
      cluster_name: cluster_name,
    };

    setDialogConfig(config);
  }, [kubeconfig_path, cluster_name]);

  return (
    <Box sx={{ mt: 2 }}>
      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="Kubernetes Config Path*"
        description="The path to the kubeconfig file."
        placeholder={Placeholders.kubeconfig_path}
        onChange={(event) => setKubeConfigPath(event.target.value)}
        value={kubeconfig_path}
      />

      <IntegrationTextInputField
        spellCheck={false}
        required={true}
        label="Cluster Name*"
        description="The name of the cluster that will be used."
        placeholder={Placeholders.cluster_name}
        onChange={(event) => setClusterName(event.target.value)}
        value={cluster_name}
      />
    </Box>
  );
};