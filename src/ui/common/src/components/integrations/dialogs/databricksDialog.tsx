import { Link } from '@mui/material';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React from 'react';

import { DatabricksConfig } from '../../../utils/integrations';
import { readOnlyFieldDisableReason, readOnlyFieldWarning } from './constants';
import { IntegrationTextInputField } from './IntegrationTextInputField';

const Placeholders: DatabricksConfig = {
  workspace_url: 'workspace_url',
  access_token: 'access_token',
  s3_instance_profile_arn: 's3_instance_profile_arn',
  instance_pool_id: 'instance_pool_id',
};

type Props = {
  onUpdateField: (field: keyof DatabricksConfig, value: string) => void;
  value?: DatabricksConfig;
  editMode: boolean;
};

export const DatabricksDialog: React.FC<Props> = ({
  onUpdateField,
  value,
  editMode,
}) => {
  return (
    <Box sx={{ mt: 2 }}>
      <Typography variant="body2">
        For more details on connecting to Databricks, please refer{' '}
        <Link href="https://docs.aqueducthq.com/integrations/compute-systems/databricks">
          the Aqueduct documentation
        </Link>
        .
      </Typography>
      <IntegrationTextInputField
        label={'Workspace URL*'}
        description={'URL of Databricks Workspace.'}
        spellCheck={false}
        required={true}
        placeholder={Placeholders.workspace_url}
        onChange={(event) => onUpdateField('workspace_url', event.target.value)}
        value={value?.workspace_url ?? ''}
        disabled={editMode}
        warning={editMode ? undefined : readOnlyFieldWarning}
        disableReason={editMode ? readOnlyFieldDisableReason : undefined}
      />

      <IntegrationTextInputField
        label={'Access Token*'}
        description={
          'The access token to connect to your Databricks Workspace.'
        }
        spellCheck={false}
        required={true}
        placeholder={Placeholders.access_token}
        onChange={(event) => onUpdateField('access_token', event.target.value)}
        value={value?.access_token ?? ''}
        disabled={editMode}
        warning={editMode ? undefined : readOnlyFieldWarning}
        disableReason={editMode ? readOnlyFieldDisableReason : undefined}
      />

      <Typography variant="body2">
        For more details on creating an S3 profile for Databricks, please see{' '}
        <Link href="https://docs.databricks.com/aws/iam/instance-profile-tutorial.html">
          the Databricks documentation
        </Link>
        .
      </Typography>

      <IntegrationTextInputField
        label={'S3 Instance Profile ARN*'}
        description={
          'The ARN of the instance profile that allows Databricks clusters to access S3.'
        }
        spellCheck={false}
        required={true}
        placeholder={Placeholders.s3_instance_profile_arn}
        onChange={(event) =>
          onUpdateField('s3_instance_profile_arn', event.target.value)
        }
        value={value?.s3_instance_profile_arn ?? ''}
        disabled={editMode}
        warning={editMode ? undefined : readOnlyFieldWarning}
        disableReason={editMode ? readOnlyFieldDisableReason : undefined}
      />

      <Typography variant="body2">
        For more details on Databricks Instance Pools, please see{' '}
        <Link href="https://docs.databricks.com/aws/iam/instance-profile-tutorial.html">
          the Databricks documentation
        </Link>
        .
      </Typography>

      <IntegrationTextInputField
        label={'Instance Pool ID'}
        description={
          'The ID of the Databricks Instance Pool that Aqueduct will run compute on.'
        }
        spellCheck={false}
        required={false}
        placeholder={Placeholders.instance_pool_id}
        onChange={(event) =>
          onUpdateField('instance_pool_id', event.target.value)
        }
        value={value?.instance_pool_id ?? ''}
      />
    </Box>
  );
};

export function isDatabricksConfigComplete(config: DatabricksConfig): boolean {
  return (
    !!config.access_token &&
    !!config.s3_instance_profile_arn &&
    !!config.workspace_url
  );
}
