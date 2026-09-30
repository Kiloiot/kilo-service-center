/**
 * Shown in place of one view or panel whose data the user's roles do not
 * cover, so a refusal never reads as an error or an empty list.
 */

import React from "react";

import { EmptyState } from "@ui/EmptyState";
import { DATA_TABLE_LAYOUT } from "@constants/app";
import { NO_ACCESS_PANEL } from "@constants/messages";
import { SecurityIcon } from "@theme/icons";

const NoAccessPanel: React.FC = () => (
  <EmptyState
    icon={<SecurityIcon />}
    title={NO_ACCESS_PANEL.TITLE}
    description={NO_ACCESS_PANEL.DESCRIPTION}
    minHeight={DATA_TABLE_LAYOUT.ERROR_MIN_HEIGHT}
  />
);

export default NoAccessPanel;
