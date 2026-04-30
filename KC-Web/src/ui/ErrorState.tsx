/**
 * Standardized Error State Component
 *
 * Displays when an error occurs in a section.
 * Uses theme tokens exclusively - no inline colors/spacing.
 */

import type { ReactNode } from 'react';
import React from 'react';

import { Box, Button, Typography } from '@mui/material';

import { ERROR_BOUNDARY } from '@constants/messages';
import { isDevelopment } from '@config/env';

export interface ErrorStateProps {
  /** Main error title */
  title?: string;
  /** Error message to display */
  message?: string;
  /** The actual error object (for dev stack traces) */
  error?: Error | null;
  /** Optional icon to display above the title */
  icon?: ReactNode;
  /** Retry action callback */
  onRetry?: () => void;
  /** Custom retry button label */
  retryLabel?: string;
  /** Optional secondary action */
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  /** Minimum height of the container */
  minHeight?: number | string;
  /** Show full error details (overrides env-based behavior) */
  showDetails?: boolean;
}

/**
 * Standardized error state component for error scenarios
 *
 * @example
 * <ErrorState
 *   title="Failed to load endpoints"
 *   message="Could not connect to the server"
 *   onRetry={refetch}
 * />
 *
 * @example
 * // With error object (shows stack in dev)
 * <ErrorState
 *   error={error}
 *   onRetry={refetch}
 * />
 */
export const ErrorState: React.FC<ErrorStateProps> = ({
  title = 'Something went wrong',
  message,
  error,
  icon,
  onRetry,
  retryLabel = 'Try again',
  secondaryAction,
  minHeight = 300,
  showDetails,
}) => {
  const displayMessage = message || error?.message || 'An unexpected error occurred';
  const shouldShowDetails = showDetails ?? isDevelopment;

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        minHeight,
        p: 4,
      }}
    >
      {icon && (
        <Box
          sx={{
            mb: 2,
            color: 'error.main',
            opacity: 0.8,
          }}
        >
          {icon}
        </Box>
      )}

      <Typography variant="h6" color="error" gutterBottom>
        {title}
      </Typography>

      <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 400, mb: 2 }}>
        {displayMessage}
      </Typography>

      {(onRetry || secondaryAction) && (
        <Box sx={{ display: 'flex', gap: 2, mt: 1 }}>
          {onRetry && (
            <Button variant="contained" onClick={onRetry}>
              {retryLabel}
            </Button>
          )}
          {secondaryAction && (
            <Button variant="outlined" onClick={secondaryAction.onClick}>
              {secondaryAction.label}
            </Button>
          )}
        </Box>
      )}

      {/* Show stack trace only in development */}
      {shouldShowDetails && error?.stack && (
        <Box sx={{ mt: 4, textAlign: 'left', maxWidth: 600, width: '100%' }}>
          <details>
            <summary style={{ cursor: 'pointer', marginBottom: 8, color: 'inherit' }}>
              <Typography variant="caption" color="text.secondary" component="span">
                {ERROR_BOUNDARY.STACK_TRACE_SUMMARY}
              </Typography>
            </summary>
            <Box
              component="pre"
              sx={{
                p: 2,
                bgcolor: 'background.paper',
                borderRadius: 1,
                overflow: 'auto',
                fontSize: '0.75rem',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                border: 1,
                borderColor: 'divider',
              }}
            >
              {error.stack}
            </Box>
          </details>
        </Box>
      )}
    </Box>
  );
};
