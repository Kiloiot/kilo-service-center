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

import { useSession } from "@contexts/SessionContext";
import { useSystem } from "@contexts/SystemContext";
import { useSignOut } from "@hooks/useSignOut";
import { formatDateTime } from "@utils/date-format";
import { ROUTES, SHORT_COMMIT_SHA_LENGTH, THEME_MODE } from "@constants/app";
import { ARIA, BRAND, USER_MENU, VERSION_INFO } from "@constants/messages";
import {
  AccountIcon,
  DarkModeIcon,
  InfoIcon,
  KeyIcon,
  LightModeIcon,
  LogoutIcon,
  OpenInNewIcon,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";
import { userMenuTriggerStyle } from "@theme/navigation";
import { useThemeMode } from "@theme/ThemeContext";

export function UserMenu() {
  const theme = useTheme();
  const navigate = useNavigate();

  // Context hooks
  const { user } = useSession();
  const { versionInfo } = useSystem();
  const { mode, toggleTheme } = useThemeMode();
  const signOut = useSignOut();

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
    await signOut();
  };

  // Build version tooltip content
  const versionTooltip = versionInfo
    ? [
        `${USER_MENU.VERSION_TOOLTIP_BUILD}: ${formatDateTime(versionInfo.buildTime)}`,
        `${USER_MENU.VERSION_TOOLTIP_COMMIT}: ${versionInfo.gitCommit?.substring(0, SHORT_COMMIT_SHA_LENGTH) || VERSION_INFO.UNKNOWN}`,
        `${USER_MENU.VERSION_TOOLTIP_BRANCH}: ${versionInfo.gitBranch || VERSION_INFO.UNKNOWN}`,
        `${USER_MENU.VERSION_TOOLTIP_SCHEMA}: ${versionInfo.schemaVersion || VERSION_INFO.UNKNOWN}`,
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
        disableRipple={false}
        sx={userMenuTriggerStyle(theme)}
      >
        <Typography
          variant="body2"
          noWrap
          sx={{
            overflow: "hidden",
            textOverflow: "ellipsis",
            maxWidth: componentSpacing.userMenu.labelMaxWidth,
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
              minWidth: componentSpacing.userMenu.menuMinWidth,
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
              {!versionInfo.isProduction && USER_MENU.DEVELOPMENT_BUILD_SUFFIX}
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
          {mode === THEME_MODE.DARK ? (
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
