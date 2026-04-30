import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import type { ActivityItem } from '@api-types/api';
import {
  useConnectionStatus,
  useDashboardAnalytics,
  useDashboardEvents,
  useDashboardStats,
  useSystemStatus,
} from '@hooks';
import {
  Box,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  IconButton,
  Paper,
  Tooltip,
  Typography,
} from '@mui/material';

import { ActivityTimeline } from '@components/common/ActivityTimeline';
import { PaginationControls } from '@components/common/PaginationControls';
import { ServiceStatusTable } from '@components/common/ServiceStatusWidget';
import {
  formatDashboardCardTrend,
  formatDashboardCountTrend,
} from '@utils/formatters';
import { PAGINATION, SERVER_ACTIVITY_CATEGORIES } from '@constants/app';
import { ACTION_REFRESH, DASHBOARD_PAGE, SYSTEM_STATUS } from '@constants/messages';
import {
  DeviceHubIcon,
  MessageIcon,
  RefreshIcon,
  RouterIcon,
  TimeIcon,
} from '@theme/icons';

interface StatusCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ReactElement;
  color: 'primary' | 'success' | 'warning' | 'error' | 'info';
  trend?: {
    value: number;
    label: string;
  };
  onClick?: () => void;
}

const StatusCard: React.FC<StatusCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  color,
  trend,
  onClick,
}) => (
  <Card sx={{ height: '100%' }}>
    <CardContent>
      <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <Box sx={{ flex: 1 }}>
          <Typography color="text.secondary" variant="body2" gutterBottom>
            {title}
          </Typography>
          <Typography variant="h4" sx={{ fontWeight: 600, mb: 0.5 }}>
            {value}
          </Typography>
          {subtitle && (
            <Typography variant="caption" color="text.secondary">
              {subtitle}
            </Typography>
          )}
          {trend && (
            <Box sx={{ mt: 1 }}>
              <Chip
                label={trend.label}
                size="small"
                sx={{
                  bgcolor: 'transparent',
                  color: 'success.main', // Always green (dashboard cards show non-negative values)
                  border: 'none',
                  fontWeight: 500,
                }}
              />
            </Box>
          )}
        </Box>
        <Box
          sx={{
            p: 1.5,
            borderRadius: 2,
            backgroundColor: `${color}.main`,
            color: 'white',
            opacity: 0.9,
            cursor: onClick ? 'pointer' : 'default',
            transition: 'all 0.2s ease',
            '&:hover': onClick
              ? {
                  opacity: 1,
                  transform: 'scale(1.05)',
                  boxShadow: 3,
                }
              : {},
          }}
          onClick={onClick}
        >
          <Box sx={{ fontSize: 32 }}>{icon}</Box>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

const Dashboard: React.FC = () => {
  const navigate = useNavigate();

  // Maintain realtime connection for updates
  useConnectionStatus();

  // React Query hooks
  const { baseStations, endpoints, isLoading, refetch } = useDashboardStats();
  const { data: analyticsData } = useDashboardAnalytics();
  // Server-side pagination state for Service Center Activity
  const [activityPage, setActivityPage] = useState(0);
  const [activityPageSize, setActivityPageSize] = useState<number>(PAGINATION.DEFAULT_PAGE_SIZE);
  const [pageTokens, setPageTokens] = useState<string[]>(['']);
  const currentPageToken = pageTokens[activityPage] || '';

  // Server activity events - filtered to exclude endpoint message traffic
  const { data: eventsData, isLoading: eventsLoading } = useDashboardEvents(
    activityPageSize,
    SERVER_ACTIVITY_CATEGORIES,
    currentPageToken
  );
  const { data: systemStatusData, isLoading: systemStatusLoading } = useSystemStatus();

  // Compute base station stats
  const baseStationStats = useMemo(() => {
    const online = baseStations.filter((bs) => bs.status === 'online').length;
    const total = baseStations.length;

    // Calculate items added in last 7 days using createdAt
    const oneWeekAgo = new Date();
    oneWeekAgo.setDate(oneWeekAgo.getDate() - 7);
    const addedLastWeek = baseStations.filter((bs) => {
      if (!bs.createdAt) return false;
      return new Date(bs.createdAt) >= oneWeekAgo;
    }).length;

    // Online percentage (0 if no basestations)
    const onlinePercent = total > 0 ? (online / total) * 100 : 0;

    return {
      total,
      online,
      offline: total - online,
      trend: {
        value: onlinePercent,
        label: formatDashboardCardTrend(
          onlinePercent,
          addedLastWeek,
          DASHBOARD_PAGE.FROM_LAST_WEEK
        ),
      },
    };
  }, [baseStations]);

  // Compute endpoint stats
  // Note: Cannot determine endpoint online/offline status (endpoints may transmit infrequently)
  const endpointStats = useMemo(() => {
    const total = endpoints.length;

    // Calculate items added in last 7 days
    const oneWeekAgo = new Date();
    oneWeekAgo.setDate(oneWeekAgo.getDate() - 7);
    const addedLastWeek = endpoints.filter((ep) => {
      if (!ep.createdAt) return false;
      return new Date(ep.createdAt) >= oneWeekAgo;
    }).length;

    return {
      total,
      trend: {
        value: addedLastWeek, // Used for chip color logic (always >= 0)
        label: formatDashboardCountTrend(addedLastWeek, DASHBOARD_PAGE.FROM_LAST_WEEK),
      },
    };
  }, [endpoints]);

  // Normalize analytics data
  // Note: totalMessages from backend is already scoped to last 24h by default
  const analytics = useMemo(
    () => ({
      totalMessages: analyticsData?.totalMessages || 0,
    }),
    [analyticsData]
  );

  // Handle server-side page change
  const handleActivityPageChange = useCallback(
    (newPage: number) => {
      if (newPage > activityPage && eventsData?.nextPageToken) {
        setPageTokens((prev) => {
          const updated = [...prev];
          updated[newPage] = eventsData.nextPageToken!;
          return updated;
        });
      }
      setActivityPage(newPage);
    },
    [activityPage, eventsData?.nextPageToken]
  );

  const handleActivityPageSizeChange = useCallback((newSize: number) => {
    setActivityPageSize(newSize);
    setActivityPage(0);
    setPageTokens(['']);
  }, []);

  // Convert dashboard events to ActivityItem[] for the unified timeline component
  const activityItems: ActivityItem[] = useMemo(
    () =>
      (eventsData?.events || []).map(
        (evt): ActivityItem => ({
          type: 'event',
          occurredAt: new Date(evt.timestamp),
          event: {
            id: evt.id,
            eventType: evt.eventType || evt.type,
            category: evt.category || '',
            severity: evt.severity,
            title: evt.title || evt.message,
            description: evt.message,
            timestamp: new Date(evt.timestamp),
            sourceName: evt.sourceName || '',
          },
        })
      ),
    [eventsData?.events]
  );

  const handleRefresh = () => {
    refetch();
  };

  const formatMessageCount = (count: number): string => {
    if (count >= 1000000) {
      return `${(count / 1000000).toFixed(1)}M`;
    } else if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  };

  return (
    <Box data-testid="dashboard-page" sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Box>
          <Typography variant="h4" component="h1">
            {DASHBOARD_PAGE.TITLE}
          </Typography>
          <Typography variant="body1" color="text.secondary">
            {DASHBOARD_PAGE.SUBTITLE}
          </Typography>
        </Box>
        <Tooltip title={ACTION_REFRESH}>
          <span>
            <IconButton size="large" onClick={handleRefresh} disabled={isLoading}>
              <RefreshIcon />
            </IconButton>
          </span>
        </Tooltip>
      </Box>

      {/* Status Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatusCard
            title={DASHBOARD_PAGE.ACTIVE_BASE_STATIONS}
            value={baseStationStats.online.toString()}
            subtitle={
              baseStationStats.offline > 0
                ? `${baseStationStats.offline} ${DASHBOARD_PAGE.OFFLINE_LABEL}`
                : DASHBOARD_PAGE.ALL_ONLINE
            }
            icon={<RouterIcon />}
            color="primary"
            trend={{
              value: baseStationStats.trend.value,
              label: baseStationStats.trend.label,
            }}
            onClick={() => navigate('/base-stations')}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatusCard
            title={DASHBOARD_PAGE.CONNECTED_ENDPOINTS}
            value={endpointStats.total.toString()}
            subtitle={
              endpointStats.total > 0
                ? DASHBOARD_PAGE.TOTAL_REGISTERED
                : DASHBOARD_PAGE.NO_ENDPOINTS
            }
            icon={<DeviceHubIcon />}
            color="primary"
            trend={{
              value: endpointStats.trend.value,
              label: endpointStats.trend.label,
            }}
            onClick={() => navigate('/endpoints')}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <StatusCard
            title={DASHBOARD_PAGE.MESSAGE_TRAFFIC}
            value={formatMessageCount(analytics.totalMessages)}
            subtitle={DASHBOARD_PAGE.LAST_24_HOURS}
            icon={<MessageIcon />}
            color="primary"
            onClick={() => navigate('/messages')}
          />
        </Grid>
      </Grid>

      {/* Alert Summary and Activity Monitor */}
      <Grid container spacing={3}>
        {/* Service Center Status */}
        <Grid size={{ xs: 12, md: 3 }}>
          <Paper sx={{ p: 3, minHeight: 360, height: '100%' }}>
            <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
              {DASHBOARD_PAGE.SERVICE_CENTER_STATUS}
            </Typography>
            {systemStatusLoading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', p: 4 }}>
                <CircularProgress size={24} />
                <Typography sx={{ ml: 1 }} color="text.secondary">
                  {SYSTEM_STATUS.LABEL_CHECKING}
                </Typography>
              </Box>
            ) : systemStatusData?.services ? (
              <ServiceStatusTable
                services={systemStatusData.services}
                timestamp={systemStatusData.timestamp}
              />
            ) : (
              <Typography color="text.secondary">{SYSTEM_STATUS.TOOLTIP_UNABLE}</Typography>
            )}
          </Paper>
        </Grid>

        {/* Network Activity Monitor */}
        <Grid size={{ xs: 12, md: 9 }}>
          <Paper
            sx={{ p: 3, minHeight: 360, height: '100%', display: 'flex', flexDirection: 'column' }}
          >
            <Box
              sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}
            >
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                {DASHBOARD_PAGE.RECENT_ACTIVITY}
              </Typography>
              <Chip
                icon={<TimeIcon />}
                label={DASHBOARD_PAGE.LIVE}
                color="success"
                size="small"
                sx={{ animation: 'pulse 2s infinite' }}
              />
            </Box>
            <Box sx={{ flex: 1, overflow: 'auto' }}>
              <ActivityTimeline variant="compact" items={activityItems} loading={eventsLoading} />
            </Box>
            <Box sx={{ mt: 'auto', pt: 2 }}>
              <PaginationControls
                page={activityPage}
                rowsPerPage={activityPageSize}
                totalCount={eventsData?.totalCount ?? 0}
                onPageChange={handleActivityPageChange}
                onRowsPerPageChange={handleActivityPageSizeChange}
              />
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* CSS Animation for pulse effect */}
      <style>
        {`
          @keyframes pulse {
            0% { opacity: 1; }
            50% { opacity: 0.6; }
            100% { opacity: 1; }
          }
        `}
      </style>
    </Box>
  );
};

export default Dashboard;
