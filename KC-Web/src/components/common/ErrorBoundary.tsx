import { Component, type ErrorInfo, type ReactNode } from "react";

import { Box, Button, Typography } from "@mui/material";

import { ERROR_BOUNDARY, UI_COMMON } from "@constants/messages";
import { isDevelopment } from "@config/env";

/**
 * Shared Error Boundary Component
 *
 * Catches JavaScript errors anywhere in the child component tree,
 * logs them, and displays a fallback UI.
 *
 * Usage:
 * - Wrap the app root for global error handling
 * - Wrap individual routes for per-route error isolation
 */

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Custom fallback component to render on error */
  fallback?: ReactNode;
  /** Callback when an error is caught (for telemetry/logging) */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  /** Identifier for this boundary (useful for telemetry) */
  boundary?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo });

    // Call optional error handler for telemetry
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  render() {
    if (this.state.hasError) {
      // Custom fallback provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback UI
      return (
        <Box
          sx={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            minHeight: "100vh",
            p: 3,
            textAlign: "center",
          }}
        >
          <Typography variant="h4" color="error" gutterBottom>
            {ERROR_BOUNDARY.TITLE}
          </Typography>
          <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
            {this.state.error?.message || UI_COMMON.ERR_GENERIC}
          </Typography>
          <Button variant="contained" onClick={this.handleReset}>
            {ERROR_BOUNDARY.RETRY_BUTTON}
          </Button>

          {/* Show stack trace only in development */}
          {isDevelopment && this.state.error?.stack && (
            <Box sx={{ mt: 4, textAlign: "left", maxWidth: 800 }}>
              <details>
                <summary style={{ cursor: "pointer", marginBottom: 8 }}>
                  {ERROR_BOUNDARY.STACK_TRACE_SUMMARY}
                </summary>
                <Box
                  component="pre"
                  sx={{
                    p: 2,
                    bgcolor: "grey.100",
                    borderRadius: 1,
                    overflow: "auto",
                    fontSize: "0.75rem",
                    whiteSpace: "pre-wrap",
                    wordBreak: "break-word",
                  }}
                >
                  {this.state.error.stack}
                </Box>
              </details>
            </Box>
          )}
        </Box>
      );
    }

    return this.props.children;
  }
}
