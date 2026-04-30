/**
 * Environment Badge Component
 *
 * Displays a visual indicator of the current environment (dev/staging/prod).
 * MUST be mounted in the App shell (AppBar/header area).
 *
 * Uses @config/env for environment detection.
 * Uses theme tokens exclusively - no inline colors/spacing.
 */

import React from 'react';

import { Chip, Tooltip } from '@mui/material';

import { ENV_LABELS, ENV_TOOLTIPS } from '@constants/app';
import { env, isProduction } from '@config/env';

type EnvType = 'development' | 'staging' | 'production';

interface EnvConfig {
  label: string;
  color: 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
  tooltip: string;
}

const ENV_CONFIGS: Record<EnvType, EnvConfig> = {
  development: {
    label: ENV_LABELS.development,
    color: 'warning',
    tooltip: ENV_TOOLTIPS.development,
  },
  staging: {
    label: ENV_LABELS.staging,
    color: 'info',
    tooltip: ENV_TOOLTIPS.staging,
  },
  production: {
    label: ENV_LABELS.production,
    color: 'success',
    tooltip: ENV_TOOLTIPS.production,
  },
};

/**
 * Detect current environment based on env config
 */
const detectEnvironment = (): EnvType => {
  if (isProduction) {
    // Check if gRPC URL suggests staging
    const grpcUrl = env.grpcUrl.toLowerCase();
    if (grpcUrl.includes('staging') || grpcUrl.includes('stage') || grpcUrl.includes('stg')) {
      return 'staging';
    }
    return 'production';
  }
  return 'development';
};

export interface EnvBadgeProps {
  /** Override environment detection (useful for testing) */
  environment?: EnvType;
  /** Only show in non-production environments */
  hideInProduction?: boolean;
  /** Custom size */
  size?: 'small' | 'medium';
}

/**
 * Environment indicator badge
 *
 * Shows the current environment (DEV/STG/PROD) with color coding.
 * Should be mounted in the AppBar/header area.
 *
 * @example
 * // In AppBar
 * <AppBar>
 *   <Toolbar>
 *     <Typography>KiloCenter</Typography>
 *     <EnvBadge />
 *   </Toolbar>
 * </AppBar>
 *
 * @example
 * // Hide in production
 * <EnvBadge hideInProduction />
 */
export const EnvBadge: React.FC<EnvBadgeProps> = ({
  environment,
  hideInProduction = false,
  size = 'small',
}) => {
  const currentEnv = environment || detectEnvironment();
  const config = ENV_CONFIGS[currentEnv];

  // Optionally hide in production
  if (hideInProduction && currentEnv === 'production') {
    return null;
  }

  // Show environment tooltip (no dev org ID - fail closed for tenant isolation)
  const tooltipContent = config.tooltip;

  return (
    <Tooltip title={tooltipContent} arrow>
      <Chip
        data-testid="env-badge"
        label={config.label}
        color={config.color}
        size={size}
        variant="filled"
        sx={(theme) => ({
          fontFamily: theme.typography.caption.fontFamily,
          fontWeight: theme.typography.fontWeightMedium,
          fontSize:
            size === 'small' ? theme.typography.caption.fontSize : theme.typography.body2.fontSize,
          height: size === 'small' ? theme.spacing(2.5) : theme.spacing(3),
          cursor: 'default',
          '& .MuiChip-label': {
            px: theme.spacing(1),
          },
        })}
      />
    </Tooltip>
  );
};
