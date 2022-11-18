import Box from '@mui/material/Box';
import React from 'react';

import { MongoDBConfig } from '../../../utils/integrations';
import { readOnlyFieldDisableReason, readOnlyFieldWarning } from './constants';
import { IntegrationTextInputField } from './IntegrationTextInputField';

const Placeholders: MongoDBConfig = {
  auth_uri: '********',
  database: 'aqueduct-db',
};

type Props = {
  onUpdateField: (field: keyof MongoDBConfig, value: string) => void;
  value?: MongoDBConfig;
  editMode: boolean;
};

export const MongoDBDialog: React.FC<Props> = ({
  onUpdateField,
  value,
  editMode,
}) => {
  return (
    <Box sx={{ mt: 2 }}>
      <IntegrationTextInputField
        label={'Uri*'}
        description={'The connection uri to your mongoDB server.'}
        spellCheck={false}
        required={true}
        placeholder={Placeholders.auth_uri}
        onChange={(event) => onUpdateField('auth_uri', event.target.value)}
        value={value?.auth_uri ?? null}
      />

      <IntegrationTextInputField
        label={'Database*'}
        description={'The name of the specific database to connect to.'}
        spellCheck={false}
        required={true}
        placeholder={Placeholders.database}
        onChange={(event) => onUpdateField('database', event.target.value)}
        value={value?.database ?? null}
        disabled={editMode}
        warning={editMode ? undefined : readOnlyFieldWarning}
        disableReason={editMode ? readOnlyFieldDisableReason : undefined}
      />
    </Box>
  );
};