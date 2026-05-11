import React from "react";
import { Link as RouterLink, useLocation } from "react-router-dom";

import {
  Box,
  Divider,
  Drawer,
  Link as MuiLink,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";

import { UserMenu } from "@components/layout/UserMenu";
import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { useSession } from "@contexts/SessionContext";
import { useSystem } from "@contexts/SystemContext";
import { useCapabilities } from "@hooks/useCapabilities";
import {
  APP_EDITION,
  DRAWER_WIDTH,
  formatPoweredByLabel,
  LOGO,
  NAV_ITEMS,
  ROUTES,
} from "@constants/app";
import { BRAND } from "@constants/messages";
import {
  BlueprintIcon,
  BusinessIcon,
  DashboardIcon,
  DeviceHubIcon,
  PeopleIcon,
  RouterIcon,
  SecurityIcon,
  VpnKeyIcon,
} from "@theme/icons";

interface NavItem {
  title: string;
  path: string;
  icon: React.ReactElement;
}

/**
 * Build navigation items based on user capabilities.
 * Server admin: sees Users, API Keys, Organizations
 * Org admin (not server admin): sees Users only
 * Regular user: sees neither
 */
const buildNavItems = (
  isServerAdmin: boolean,
  isOrgAdmin: boolean,
  showOrgs: boolean,
): NavItem[] => [
  { title: NAV_ITEMS.DASHBOARD, path: ROUTES.HOME, icon: <DashboardIcon /> },
  {
    title: NAV_ITEMS.BASE_STATIONS,
    path: ROUTES.BASE_STATIONS,
    icon: <RouterIcon />,
  },
  {
    title: NAV_ITEMS.ENDPOINTS,
    path: ROUTES.ENDPOINTS,
    icon: <DeviceHubIcon />,
  },
  {
    title: NAV_ITEMS.BLUEPRINTS,
    path: ROUTES.BLUEPRINTS,
    icon: <BlueprintIcon />,
  },
  // Users & Roles: visible to server admin and org admin
  ...(isServerAdmin || isOrgAdmin
    ? [{ title: NAV_ITEMS.USERS, path: ROUTES.USERS, icon: <PeopleIcon /> }]
    : []),
  // API Keys: server admin only (both CE and ECE)
  ...(isServerAdmin
    ? [
        {
          title: NAV_ITEMS.API_KEYS,
          path: ROUTES.API_KEYS,
          icon: <VpnKeyIcon />,
        },
      ]
    : []),
  // Organizations: server admin only, ECE only
  ...(isServerAdmin && showOrgs
    ? [
        {
          title: NAV_ITEMS.ORGANIZATIONS,
          path: ROUTES.ORGANIZATIONS,
          icon: <BusinessIcon />,
        },
      ]
    : []),
  {
    title: NAV_ITEMS.CERTIFICATES,
    path: ROUTES.CERTIFICATES,
    icon: <SecurityIcon />,
  },
];

interface AppNavigationProps {
  mobileOpen: boolean;
  handleDrawerToggle: () => void;
}

const AppNavigation: React.FC<AppNavigationProps> = ({
  mobileOpen,
  handleDrawerToggle,
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const location = useLocation();
  const { isHydrated } = useSession();
  const { isServerAdmin, isOrgAdmin } = useCapabilities();
  const { versionInfo } = useSystem();
  const { isEnabled } = useFeatureFlags();
  const showOrgs = isEnabled("enterprise_organizations");

  // Gate nav rendering until hydration completes to prevent flicker
  // (user may be null during first load before storage is read)
  const showServerAdmin = isHydrated && isServerAdmin;
  const showOrgAdmin = isHydrated && isOrgAdmin;
  const navItems = React.useMemo(
    () => buildNavItems(showServerAdmin, showOrgAdmin, showOrgs),
    [showServerAdmin, showOrgAdmin, showOrgs],
  );

  // Mode-aware navigation colors
  const isDark = theme.palette.mode === "dark";
  const logoPath = isDark ? LOGO.DARK_PATH : LOGO.LIGHT_PATH;
  // In dark mode, use light gray for icon visibility; in light mode, use dark blue primary
  const iconColor = isDark
    ? theme.palette.grey[100]
    : theme.palette.primary.main;
  // Hover: medium gray per mode
  const hoverBg = isDark ? theme.palette.grey[700] : theme.palette.grey[300];
  // Selected: distinct from hover - lighter in dark mode for contrast with white text
  const selectedBg = isDark ? theme.palette.grey[600] : theme.palette.grey[200];

  const drawer = (
    <Box
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        backgroundColor: theme.palette.background.default,
        color: theme.palette.text.primary,
      }}
    >
      <Toolbar
        sx={{
          ...theme.mixins.toolbar,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          minHeight: "auto",
          py: theme.spacing(1.5),
        }}
      >
        <img
          src={logoPath}
          alt={LOGO.ALT}
          style={{ width: LOGO.WIDTH, height: "auto" }}
        />
      </Toolbar>
      <Divider />
      <Box sx={{ flex: 1, overflow: "auto", pt: 3 }}>
        <List>
          {navItems.map((item) => (
            <ListItem key={item.path} disablePadding>
              <ListItemButton
                component={RouterLink}
                to={item.path}
                selected={location.pathname === item.path}
                onClick={() => isMobile && handleDrawerToggle()}
                sx={{
                  "&:hover": {
                    backgroundColor: hoverBg,
                  },
                  "&.Mui-selected": {
                    backgroundColor: selectedBg,
                    "&:hover": {
                      backgroundColor: hoverBg,
                    },
                  },
                }}
              >
                <ListItemIcon sx={{ color: iconColor }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText primary={item.title} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Box>

      {/* Edition and disclosure footer */}
      <Box
        sx={{ px: 2, py: 1.5, borderTop: `1px solid ${theme.palette.divider}` }}
      >
        <Typography variant="caption" display="block" color="text.secondary">
          {versionInfo?.edition ?? APP_EDITION}
        </Typography>
        <Typography
          variant="caption"
          display="block"
          color="text.secondary"
          sx={{ mb: 0.5 }}
        >
          {formatPoweredByLabel(versionInfo?.edition ?? APP_EDITION)}
        </Typography>
        <Box sx={{ display: "flex", gap: 1.5 }}>
          <MuiLink
            href={versionInfo?.sourceUrl}
            target="_blank"
            rel="noopener"
            variant="caption"
          >
            {BRAND.SOURCE}
          </MuiLink>
          <MuiLink
            href={versionInfo?.documentationUrl}
            target="_blank"
            rel="noopener"
            variant="caption"
          >
            {BRAND.DOCUMENTATION}
          </MuiLink>
          <MuiLink
            href={versionInfo?.licenseUrl}
            target="_blank"
            rel="noopener"
            variant="caption"
          >
            {BRAND.LICENSE}
          </MuiLink>
        </Box>
      </Box>

      {/* User menu footer */}
      <Box
        sx={{
          p: theme.spacing(2),
          borderTop: `1px solid ${theme.palette.divider}`,
          backgroundColor: theme.palette.background.default,
        }}
      >
        <UserMenu />
      </Box>
    </Box>
  );

  return (
    <Box
      component="nav"
      sx={{ width: { sm: DRAWER_WIDTH }, flexShrink: { sm: 0 } }}
    >
      {/* Mobile drawer */}
      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={handleDrawerToggle}
        ModalProps={{
          keepMounted: true,
        }}
        sx={{
          display: { xs: "block", sm: "none" },
          "& .MuiDrawer-paper": {
            boxSizing: "border-box",
            width: DRAWER_WIDTH,
            backgroundColor: theme.palette.background.paper,
            color: theme.palette.text.primary,
          },
        }}
      >
        {drawer}
      </Drawer>
      {/* Desktop drawer */}
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: "none", sm: "block" },
          "& .MuiDrawer-paper": {
            boxSizing: "border-box",
            width: DRAWER_WIDTH,
            backgroundColor: theme.palette.background.paper,
            color: theme.palette.text.primary,
          },
        }}
        open
      >
        {drawer}
      </Drawer>
    </Box>
  );
};

export default AppNavigation;
