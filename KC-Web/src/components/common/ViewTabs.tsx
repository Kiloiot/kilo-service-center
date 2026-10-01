/**
 * Tabs over a fixed list of views; only the active view is rendered, so its
 * queries run only while it is on screen.
 */

import { type ReactElement, type ReactNode, useId, useState } from "react";

import { Box } from "@mui/material";

import { TabBar } from "./TabBar";
import { TabPanel } from "./TabPanel";

interface ViewTabsProps<V extends string> {
  views: readonly V[];
  labels: Record<V, string>;
  icons?: Partial<Record<V, ReactElement>>;
  ariaLabel: string;
  children: (view: V) => ReactNode;
}

export function ViewTabs<V extends string>({
  views,
  labels,
  icons,
  ariaLabel,
  children,
}: ViewTabsProps<V>) {
  const idPrefix = useId();
  const [active, setActive] = useState<V>(views[0]);

  return (
    <Box>
      <TabBar
        value={active}
        onChange={setActive}
        items={views.map((view) => ({
          value: view,
          label: labels[view],
          icon: icons?.[view],
        }))}
        ariaLabel={ariaLabel}
        idPrefix={idPrefix}
      />
      <TabPanel idPrefix={idPrefix} value={active}>
        {children(active)}
      </TabPanel>
    </Box>
  );
}
