import React, { useMemo } from "react";
import { BrowserRouter as Router, useLocation } from "react-router-dom";

import { useRealtimeUpdates } from "@hooks";
import {
  AppBar,
  Box,
  IconButton,
  Toolbar,
  Typography,
  useTheme,
} from "@mui/material";
import { AppRouter, AuthGuard } from "@router";
import { QueryClientProvider } from "@tanstack/react-query";
import { EnvBadge } from "@ui";

import GlobalLoader from "@components/common/GlobalLoader";
import AlertBanner from "@components/layout/AlertBanner";
import AppNavigation from "@components/layout/AppNavigation";
import OrgBadge from "@components/layout/OrgBadge";
import type { FeatureFlags } from "@contexts/FeatureFlagContext";
import { FeatureFlagProvider } from "@contexts/FeatureFlagContext";
import { FeedbackProvider } from "@contexts/feedback";
import { FiltersProvider } from "@contexts/filters";
import { OrganizationProvider } from "@contexts/OrganizationContext";
import { SessionProvider } from "@contexts/SessionContext";
import { SystemProvider, useSystem } from "@contexts/SystemContext";
import { APP_TITLE, DRAWER_WIDTH, EDITION_CODE, ROUTES } from "@constants/app";
import { ARIA } from "@constants/messages";
import { isDevelopment } from "@config/env";
import { queryClient } from "@config/query-client";
import { MenuIcon } from "@theme/icons";

import "leaflet/dist/leaflet.css";

const ReactQueryDevtools = isDevelopment
  ? React.lazy(() =>
      import("@tanstack/react-query-devtools").then((m) => ({
        default: m.ReactQueryDevtools,
      })),
    )
  : null;

// Public routes that should render without the main layout (header, navigation)
const PUBLIC_ROUTES = [ROUTES.LOGIN, ROUTES.REGISTER, ROUTES.AUTH_CALLBACK];

const AppContent: React.FC = () => {
  const [mobileOpen, setMobileOpen] = React.useState(false);
  const theme = useTheme();
  const location = useLocation();

  // Check if current route is a public route (login, auth callback)
  const isPublicRoute = PUBLIC_ROUTES.some((path) =>
    location.pathname.startsWith(path),
  );

  // Wire realtime connection + cache invalidation globally
  useRealtimeUpdates();

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  // Public routes render without the main layout (full-page login/callback)
  if (isPublicRoute) {
    return (
      <React.Suspense fallback={<GlobalLoader />}>
        <AppRouter />
      </React.Suspense>
    );
  }

  return (
    <Box sx={{ display: "flex" }}>
      {/* App Bar */}
      <AppBar
        position="fixed"
        sx={{
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
          ml: { sm: `${DRAWER_WIDTH}px` },
          borderRadius: 0,
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label={ARIA.OPEN_DRAWER}
            edge="start"
            onClick={handleDrawerToggle}
            sx={{ mr: theme.spacing(2), display: { sm: "none" } }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            {APP_TITLE}
          </Typography>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: theme.spacing(2),
            }}
          >
            {/* Environment Badge */}
            <EnvBadge hideInProduction />
            <Box sx={{ display: { xs: "none", sm: "inline-flex" } }}>
              <OrgBadge />
            </Box>
          </Box>
        </Toolbar>
      </AppBar>

      {/* Navigation */}
      <AppNavigation
        mobileOpen={mobileOpen}
        handleDrawerToggle={handleDrawerToggle}
      />

      {/* Main content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: theme.spacing(3),
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
        }}
      >
        <Toolbar sx={theme.mixins.toolbar} />

        {/* Alert Banner */}
        <AlertBanner />

        {/* Centralized Router with Feature Protection */}
        <React.Suspense fallback={<GlobalLoader />}>
          <AppRouter />
        </React.Suspense>
      </Box>
    </Box>
  );
};

/**
 * Derives edition-aware feature flag overrides from system version info.
 * CE: enterprise_organizations=false (default), ECE: enterprise_organizations=true
 */
const EditionAwareApp: React.FC = () => {
  const { versionInfo, loading: systemLoading } = useSystem();

  const isEnterprise = versionInfo?.editionCode === EDITION_CODE.ENTERPRISE;

  const flagOverrides: Partial<FeatureFlags> | undefined = useMemo(
    () =>
      systemLoading ? undefined : { enterprise_organizations: isEnterprise },
    [systemLoading, isEnterprise],
  );

  return (
    <FeatureFlagProvider overrides={flagOverrides}>
      <Router>
        <AuthGuard>
          <AppContent />
        </AuthGuard>
      </Router>
    </FeatureFlagProvider>
  );
};

const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <FeedbackProvider>
        <SystemProvider>
          {/* SessionProvider MUST wrap OrganizationProvider so useSession() works in OrganizationContext */}
          <SessionProvider>
            <OrganizationProvider>
              {/* FiltersProvider needs OrganizationContext for org-scoped storage */}
              <FiltersProvider>
                <EditionAwareApp />
              </FiltersProvider>
            </OrganizationProvider>
          </SessionProvider>
        </SystemProvider>
      </FeedbackProvider>
      {isDevelopment && ReactQueryDevtools && (
        <React.Suspense fallback={null}>
          <ReactQueryDevtools initialIsOpen={false} />
        </React.Suspense>
      )}
    </QueryClientProvider>
  );
};

export default App;
