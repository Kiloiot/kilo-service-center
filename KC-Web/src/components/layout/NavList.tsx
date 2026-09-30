import { Link as RouterLink, useLocation } from "react-router-dom";

import {
  List,
  ListItem,
  ListItemButton,
  listItemButtonClasses,
  ListItemIcon,
  ListItemText,
  type Theme,
  useTheme,
} from "@mui/material";

import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { FEATURE_FLAG, THEME_MODE } from "@constants/app";
import { highlightSurface, listItemStates } from "@theme/highlight";

import { buildNavItems } from "./navItems";

/** Grey hover and selection surfaces with regular text, instead of the theme highlight. */
const navItemStyles = (theme: Theme) => {
  const isDark = theme.palette.mode === THEME_MODE.DARK;
  const surface = (main: string) =>
    highlightSurface({
      main,
      contrastText: theme.palette.text.primary,
      secondaryText: theme.palette.text.secondary,
    });
  const states = listItemStates(listItemButtonClasses);
  return {
    iconColor: isDark ? theme.palette.grey[100] : theme.palette.primary.main,
    item: {
      [states.selected]: surface(
        isDark ? theme.palette.grey[600] : theme.palette.grey[200],
      ),
      [states.hovered]: surface(
        isDark ? theme.palette.grey[700] : theme.palette.grey[300],
      ),
    },
  };
};

interface NavListProps {
  onNavigate: () => void;
}

/** The navigation entries the signed-in user's roles allow. */
export function NavList({ onNavigate }: NavListProps) {
  const theme = useTheme();
  const { pathname } = useLocation();
  const { isHydrated } = useSession();
  const { can, rolesLoaded } = useCapabilities();
  const { isEnabled } = useFeatureFlags();
  const styles = navItemStyles(theme);

  // Gate nav rendering until the session and roles are known to prevent flicker
  const items =
    isHydrated && rolesLoaded
      ? buildNavItems(can, isEnabled(FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS))
      : [];

  return (
    <List>
      {items.map(({ path, title, icon: Icon }) => (
        <ListItem key={path} disablePadding>
          <ListItemButton
            component={RouterLink}
            to={path}
            selected={pathname === path}
            onClick={onNavigate}
            sx={styles.item}
          >
            <ListItemIcon sx={{ color: styles.iconColor }}>
              <Icon />
            </ListItemIcon>
            <ListItemText primary={title} />
          </ListItemButton>
        </ListItem>
      ))}
    </List>
  );
}
