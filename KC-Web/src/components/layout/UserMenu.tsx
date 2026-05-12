/**
 * UserMenu Component
 *
 * Displays authenticated user email as trigger with dropdown menu containing:
 * - Version header row (with build info tooltip)
 * - Theme toggle
 * - Change password (if user.hasPassword === true)
 * - Logout
 */

import { type MouseEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthSettings } from "@hooks";
import {
  Box,
  Button,
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
  Typography,
  useTheme,
} from "@mui/material";

import { apiService } from "@services/api";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useSystem } from "@contexts/SystemContext";
import { storageService } from "@utils/storage";
import { ROUTES, STORAGE_KEYS } from "@constants/app";
import { ARIA, BRAND, USER_MENU } from "@constants/messages";
import { resetQueryClient } from "@config/query-client";
import {
  AccountIcon,
  DarkModeIcon,
  InfoIcon,
  KeyIcon,
  LightModeIcon,
  LogoutIcon,
  OpenInNewIcon,
} from "@theme/icons";
import { useThemeMode } from "@theme/ThemeContext";

export function UserMenu() {
  const theme = useTheme();
  const navigate = useNavigate();

  // Context hooks
  const { user, clearSession } = useSession();
  const { clearOrganization } = useOrganization();
  const { versionInfo } = useSystem();
  const { mode, toggleTheme } = useThemeMode();
  const { data: authSettings } = useAuthSettings();

  // Menu state
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleThemeToggle = () => {
    toggleTheme();
    handleClose();
  };

  const handleChangePassword = () => {
    handleClose();
    navigate(ROUTES.MY_PASSWORD);
  };

  const handleLogout = async () => {
    handleClose();

    const refreshToken = storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN);

    // Only call logout API if BOTH conditions are true:
    // 1. authSettings.refresh_token_enabled === true
    // 2. refreshToken exists in storage
    // Otherwise the backend returns 501 Not Implemented
    if (authSettings?.refresh_token_enabled === true && refreshToken) {
      try {
        await apiService.logout(refreshToken);
      } catch {
        // Ignore errors - still proceed with local cleanup
      }
    }

    // Clear all state regardless of API call
    clearSession();
    clearOrganization();
    storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    resetQueryClient();

    // Redirect to provider logout URL if configured, otherwise login page
    const logoutUrl =
      authSettings?.oidc?.logout_url || authSettings?.oauth2?.logout_url;
    if (logoutUrl) {
      window.location.href = logoutUrl;
    } else {
      navigate(ROUTES.LOGIN);
    }
  };

  // Build version tooltip content
  const versionTooltip = versionInfo
    ? [
        `${USER_MENU.VERSION_TOOLTIP_BUILD}: ${new Date(versionInfo.buildTime).toLocaleString()}`,
        `${USER_MENU.VERSION_TOOLTIP_COMMIT}: ${versionInfo.gitCommit?.substring(0, 8) || "unknown"}`,
        `${USER_MENU.VERSION_TOOLTIP_BRANCH}: ${versionInfo.gitBranch || "unknown"}`,
        `${USER_MENU.VERSION_TOOLTIP_SCHEMA}: ${versionInfo.schemaVersion || "unknown"}`,
      ].join("\n")
    : "";

  if (!user) {
    return null;
  }

  return (
    <Box>
      <Button
        id="user-menu-button"
        aria-controls={open ? "user-menu" : undefined}
        aria-haspopup="true"
        aria-expanded={open ? "true" : undefined}
        aria-label={ARIA.USER_MENU_TOGGLE}
        onClick={handleClick}
        startIcon={<AccountIcon />}
        sx={{
          width: "100%",
          justifyContent: "flex-start",
          textTransform: "none",
          color: theme.palette.text.primary,
          px: 1.5,
          py: 1,
          "&:hover": {
            backgroundColor: theme.palette.action.hover,
          },
        }}
      >
        <Typography
          variant="body2"
          noWrap
          sx={{
            overflow: "hidden",
            textOverflow: "ellipsis",
            maxWidth: 160,
          }}
        >
          {user.email}
        </Typography>
      </Button>

      <Menu
        id="user-menu"
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        MenuListProps={{
          "aria-labelledby": "user-menu-button",
          "aria-label": ARIA.USER_MENU,
        }}
        anchorOrigin={{
          vertical: "top",
          horizontal: "left",
        }}
        transformOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        slotProps={{
          paper: {
            sx: {
              minWidth: 200,
            },
          },
        }}
      >
        {/* Version header row */}
        {versionInfo && (
          <Box
            sx={{
              px: 2,
              py: 1,
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              borderBottom: `1px solid ${theme.palette.divider}`,
            }}
          >
            <Typography variant="caption" color="text.secondary">
              {versionInfo.version}
              {!versionInfo.isProduction && " (dev)"}
            </Typography>
            <Tooltip
              title={
                <pre style={{ margin: 0, whiteSpace: "pre-wrap" }}>
                  {versionTooltip}
                </pre>
              }
            >
              <InfoIcon
                fontSize="small"
                color="action"
                sx={{ cursor: "pointer" }}
              />
            </Tooltip>
          </Box>
        )}

        {/* External documentation and source links */}
        <MenuItem
          component="a"
          href={versionInfo?.documentationUrl}
          target="_blank"
          rel="noopener"
          dense
        >
          <ListItemIcon>
            <OpenInNewIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary={BRAND.DOCUMENTATION} />
        </MenuItem>
        <MenuItem
          component="a"
          href={versionInfo?.sourceUrl}
          target="_blank"
          rel="noopener"
          dense
        >
          <ListItemIcon>
            <OpenInNewIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary={BRAND.SOURCE} />
        </MenuItem>
        <MenuItem
          component="a"
          href={versionInfo?.licenseUrl}
          target="_blank"
          rel="noopener"
          dense
        >
          <ListItemIcon>
            <OpenInNewIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary={BRAND.LICENSE} />
        </MenuItem>
        <Divider />

        {/* Theme toggle */}
        <MenuItem onClick={handleThemeToggle}>
          {mode === "dark" ? (
            <>
              <LightModeIcon sx={{ mr: 1.5 }} fontSize="small" />
              {USER_MENU.THEME_LIGHT}
            </>
          ) : (
            <>
              <DarkModeIcon sx={{ mr: 1.5 }} fontSize="small" />
              {USER_MENU.THEME_DARK}
            </>
          )}
        </MenuItem>

        {/* Change password - only if user has a password set */}
        {user.hasPassword && (
          <MenuItem onClick={handleChangePassword}>
            <KeyIcon sx={{ mr: 1.5 }} fontSize="small" />
            {USER_MENU.CHANGE_PASSWORD}
          </MenuItem>
        )}

        <Divider />

        {/* Logout */}
        <MenuItem onClick={handleLogout}>
          <LogoutIcon sx={{ mr: 1.5 }} fontSize="small" />
          {USER_MENU.LOGOUT}
        </MenuItem>
      </Menu>
    </Box>
  );
}
