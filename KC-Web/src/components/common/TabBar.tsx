/**
 * The one tab bar: a controlled row of tabs over a divider. The owner keeps
 * the active value, in state or in the URL.
 */

import type { ReactElement } from "react";

import { Box, Tab, Tabs } from "@mui/material";

import { tabId, tabPanelId } from "./tabIds";

export interface TabBarItem<V extends string> {
  value: V;
  label: string;
  icon?: ReactElement;
}

interface TabBarProps<V extends string> {
  value: V;
  onChange: (value: V) => void;
  items: readonly TabBarItem<V>[];
  ariaLabel: string;
  idPrefix: string;
}

export function TabBar<V extends string>({
  value,
  onChange,
  items,
  ariaLabel,
  idPrefix,
}: TabBarProps<V>) {
  return (
    <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
      <Tabs
        value={value}
        onChange={(_, next: V) => onChange(next)}
        aria-label={ariaLabel}
        variant="scrollable"
        scrollButtons="auto"
      >
        {items.map((item) => (
          <Tab
            key={item.value}
            value={item.value}
            label={item.label}
            icon={item.icon}
            iconPosition="start"
            id={tabId(idPrefix, item.value)}
            aria-controls={tabPanelId(idPrefix, item.value)}
          />
        ))}
      </Tabs>
    </Box>
  );
}
