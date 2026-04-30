import React from 'react';

import { useConnectionStatus } from '@hooks';
import { Alert, Collapse } from '@mui/material';

import { STATUS_OFFLINE, STATUS_RECONNECTING } from '@constants/messages';

/**
 * Alert Banner Component
 * Displays system-wide alerts in application shell
 * Includes automatic connection status display
 */

interface AlertBannerProps {
  show?: boolean;
  severity?: 'error' | 'warning' | 'info' | 'success';
  message?: string;
  showConnectionStatus?: boolean;
}

const AlertBanner: React.FC<AlertBannerProps> = ({
  show = false,
  severity = 'info',
  message = '',
  showConnectionStatus = false,
}) => {
  const { status, isConnected } = useConnectionStatus();

  // Connection status banner (shown when disconnected/reconnecting)
  const showConnectionBanner = showConnectionStatus && !isConnected;
  const connectionSeverity = status === 'reconnecting' ? 'warning' : 'error';
  const connectionMessage = status === 'reconnecting' ? STATUS_RECONNECTING : STATUS_OFFLINE;

  return (
    <>
      {/* Connection status banner */}
      <Collapse in={showConnectionBanner}>
        <Alert severity={connectionSeverity} sx={{ mb: 2 }}>
          {connectionMessage}
        </Alert>
      </Collapse>

      {/* Custom alert banner */}
      <Collapse in={show && message.length > 0}>
        <Alert severity={severity} sx={{ mb: 2 }}>
          {message}
        </Alert>
      </Collapse>
    </>
  );
};

export default AlertBanner;
