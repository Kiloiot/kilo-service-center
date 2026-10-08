import type React from "react";

import {
  Box,
  Divider,
  Drawer,
  Toolbar,
  useMediaQuery,
  useTheme,
} from "@mui/material";

import { ProductLogo } from "@components/common/ProductLogo";
import { UserMenu } from "@components/layout/UserMenu";
import { DRAWER_WIDTH, LOGO } from "@constants/app";

import { DisclosureLinks } from "./DisclosureLinks";
import { NavList } from "./NavList";

function NavLogo() {
  const theme = useTheme();

  return (
    <Toolbar
      sx={{
        ...theme.mixins.toolbar,
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "auto",
        py: 1.5,
      }}
    >
      <ProductLogo width={LOGO.NAV_WIDTH} />
    </Toolbar>
  );
}

/** Logo, navigation entries, edition disclosure and user menu, top to bottom. */
function DrawerContent({ onNavigate }: { onNavigate: () => void }) {
  return (
    <Box
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        bgcolor: "background.default",
        color: "text.primary",
      }}
    >
      <NavLogo />
      <Divider />
      <Box sx={{ flex: 1, overflow: "auto", pt: 3 }}>
        <NavList onNavigate={onNavigate} />
      </Box>
      <DisclosureLinks />
      <Box
        sx={{
          p: 2,
          borderTop: 1,
          borderColor: "divider",
          bgcolor: "background.default",
        }}
      >
        <UserMenu />
      </Box>
    </Box>
  );
}

interface AppNavigationProps {
  mobileOpen: boolean;
  handleDrawerToggle: () => void;
}

const DRAWER_PAPER_SX = {
  "& .MuiDrawer-paper": {
    boxSizing: "border-box",
    width: DRAWER_WIDTH,
    bgcolor: "background.paper",
    color: "text.primary",
  },
};

/** A temporary drawer on phones and a permanent one from the small breakpoint up. */
const AppNavigation: React.FC<AppNavigationProps> = ({
  mobileOpen,
  handleDrawerToggle,
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const content = (
    <DrawerContent onNavigate={() => isMobile && handleDrawerToggle()} />
  );

  return (
    <Box
      component="nav"
      sx={{ width: { sm: DRAWER_WIDTH }, flexShrink: { sm: 0 } }}
    >
      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={handleDrawerToggle}
        ModalProps={{ keepMounted: true }}
        sx={{ display: { xs: "block", sm: "none" }, ...DRAWER_PAPER_SX }}
      >
        {content}
      </Drawer>
      <Drawer
        variant="permanent"
        sx={{ display: { xs: "none", sm: "block" }, ...DRAWER_PAPER_SX }}
        open
      >
        {content}
      </Drawer>
    </Box>
  );
};

export default AppNavigation;
