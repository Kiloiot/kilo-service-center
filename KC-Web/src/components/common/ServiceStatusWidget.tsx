/**
 * Service Status Widget
 *
 * Reusable service health status table component.
 * Renders Status/Service/Latency columns with a "Last checked" timestamp.
 */

import {
  Box,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';

import { SYSTEM_STATUS } from '@constants/messages';
import { ErrorIcon, SuccessIcon } from '@theme/icons';

/**
 * Props for ServiceStatusTable component
 */
interface ServiceStatusTableProps {
  /** Array of services to display - rendered in order received (no sorting) */
  services: Array<{ name: string; healthy: boolean; latencyMs: number; error?: string }>;
  /** Timestamp of last status check */
  timestamp?: string;
}

/**
 * Reusable Service Status Table Component
 *
 * Renders the system status table with Status/Service/Latency columns
 * and "Last checked" timestamp. Used by the Dashboard.
 */
export function ServiceStatusTable({ services, timestamp }: ServiceStatusTableProps) {
  return (
    <>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{SYSTEM_STATUS.LABEL_STATUS}</TableCell>
            <TableCell>{SYSTEM_STATUS.LABEL_SERVICE}</TableCell>
            <TableCell align="right">{SYSTEM_STATUS.LABEL_LATENCY}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {services.map((service) => (
            <TableRow key={service.name}>
              <TableCell>
                {service.healthy ? (
                  <SuccessIcon color="success" fontSize="small" />
                ) : (
                  <Tooltip title={service.error || ''}>
                    <ErrorIcon color="error" fontSize="small" />
                  </Tooltip>
                )}
              </TableCell>
              <TableCell>
                <Typography variant="body2">{service.name}</Typography>
                {service.error && (
                  <Typography variant="caption" color="error">
                    {service.error}
                  </Typography>
                )}
              </TableCell>
              <TableCell align="right">
                <Typography variant="body2">{service.latencyMs}ms</Typography>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Box sx={{ mt: 2, textAlign: 'center' }}>
        <Typography variant="caption" color="text.secondary">
          {SYSTEM_STATUS.LABEL_LAST_CHECKED}:{' '}
          {timestamp ? new Date(timestamp).toLocaleTimeString() : 'N/A'}
        </Typography>
      </Box>
    </>
  );
}
