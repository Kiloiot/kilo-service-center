/**
 * Standardized Empty State Component
 *
 * Displays when no data is available in a section.
 * Uses theme tokens exclusively - no inline colors/spacing.
 */

import type { ReactNode } from 'react';
import React from 'react';

import { Box, Button, Typography } from '@mui/material';

export interface EmptyStateProps {
  /** Main title text */
  title: string;
  /** Optional description text */
  description?: string;
  /** Optional icon to display above the title */
  icon?: ReactNode;
  /** Optional primary action button */
  action?: {
    label: string;
    onClick: () => void;
  };
  /** Optional secondary action button */
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  /** Minimum height of the container */
  minHeight?: number | string;
}

/**
 * Standardized empty state component for no-data scenarios
 *
 * @example
 * <EmptyState
 *   title="No endpoints found"
 *   description="Create your first endpoint to get started"
 *   icon={<DevicesIcon sx={{ fontSize: 64 }} />}
 *   action={{ label: "Create Endpoint", onClick: handleCreate }}
 * />
 */
export const EmptyState: React.FC<EmptyStateProps> = ({
  title,
  description,
  icon,
  action,
  secondaryAction,
  minHeight = 300,
}) => {
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
            color: 'text.secondary',
            opacity: 0.6,
          }}
        >
          {icon}
        </Box>
      )}

      <Typography variant="h6" color="text.primary" gutterBottom>
        {title}
      </Typography>

      {description && (
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ maxWidth: 400, mb: action ? 3 : 0 }}
        >
          {description}
        </Typography>
      )}

      {(action || secondaryAction) && (
        <Box sx={{ display: 'flex', gap: 2, mt: 2 }}>
          {action && (
            <Button variant="contained" onClick={action.onClick}>
              {action.label}
            </Button>
          )}
          {secondaryAction && (
            <Button variant="outlined" onClick={secondaryAction.onClick}>
              {secondaryAction.label}
            </Button>
          )}
        </Box>
      )}
    </Box>
  );
};
