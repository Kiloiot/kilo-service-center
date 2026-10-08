/** The panel of the active tab, labelled by its tab in the TabBar. */

import type { ReactNode } from "react";

import { Box } from "@mui/material";

import { componentSpacing } from "@theme/index";

import { tabId, tabPanelId } from "./tabIds";

interface TabPanelProps {
  idPrefix: string;
  value: string;
  children: ReactNode;
}

export function TabPanel({ idPrefix, value, children }: TabPanelProps) {
  return (
    <Box
      role="tabpanel"
      id={tabPanelId(idPrefix, value)}
      aria-labelledby={tabId(idPrefix, value)}
      sx={{ pt: componentSpacing.tabs.panelPt }}
    >
      {children}
    </Box>
  );
}
