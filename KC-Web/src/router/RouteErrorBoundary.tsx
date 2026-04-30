/**
 * Route Error Boundary
 *
 * Per-route error boundary for isolating route-level errors.
 * Uses the shared ErrorState component for consistent UI.
 */

import { Component, type ErrorInfo, type ReactNode } from 'react';

import { Box } from '@mui/material';
import { ErrorState } from '@ui';

import { logger } from '@utils/logger';
import { ERROR_BOUNDARY, UI_COMMON } from '@constants/messages';
import { isDevelopment } from '@config/env';

interface RouteErrorBoundaryProps {
  children: ReactNode;
  /** Route name for error reporting */
  routeName?: string;
  /** Callback when error is caught */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface RouteErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Per-route error boundary component
 *
 * Wraps each route to catch and handle route-level errors.
 * Shows ErrorState UI with retry option.
 */
export class RouteErrorBoundary extends Component<
  RouteErrorBoundaryProps,
  RouteErrorBoundaryState
> {
  constructor(props: RouteErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): Partial<RouteErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Log error in development
    if (isDevelopment) {
      logger.error('[RouteErrorBoundary]', this.props.routeName || 'Unknown route', error);
    }

    // Call optional error handler for telemetry
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  handleGoHome = () => {
    window.location.href = '/';
  };

  render() {
    if (this.state.hasError) {
      return (
        <Box sx={{ p: 3, pt: 10 }}>
          <ErrorState
            title={ERROR_BOUNDARY.ROUTE_ERROR_TITLE}
            message={this.state.error?.message || UI_COMMON.ERR_LOAD_FAILED}
            error={this.state.error}
            onRetry={this.handleRetry}
            retryLabel={ERROR_BOUNDARY.REFRESH_BUTTON}
            secondaryAction={{
              label: UI_COMMON.TITLE_DASHBOARD,
              onClick: this.handleGoHome,
            }}
            minHeight={400}
          />
        </Box>
      );
    }

    return this.props.children;
  }
}
