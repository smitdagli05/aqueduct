import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React from 'react';

import { Integration, SlackConfig } from '../../../utils/integrations';

type Props = {
  integration: Integration;
};

export const SlackCard: React.FC<Props> = ({ integration }) => {
  const config = integration.config as SlackConfig;
  const channels = JSON.parse(config.channels_serialized) as string[];
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <Typography variant="body2">
        {channels.length > 1 ? (
          <strong>Channel: </strong>
        ) : (
          <strong>Channels:</strong>
        )}{' '}
        {channels.join(', ')}
      </Typography>
      <Typography variant="body2">
        <strong>Level: </strong>
        {config.level.toUpperCase()}
      </Typography>
    </Box>
  );
};