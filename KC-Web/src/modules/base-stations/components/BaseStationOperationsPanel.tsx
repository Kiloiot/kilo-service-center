import React from "react";

import { Grid } from "@mui/material";

import { componentSpacing } from "@theme/index";

import { BaseStationAvailabilityCard } from "./BaseStationAvailabilityCard";
import { BaseStationCertificateCard } from "./BaseStationCertificateCard";
import { BaseStationPingCard } from "./BaseStationPingCard";

interface BaseStationOperationsPanelProps {
  bsEui: string;
  online: boolean;
  certificateExpiresAt?: string;
  certificateFingerprint?: string;
  lastHandshake?: string;
}

/** Base station detail health, certificate and action cards (NAV 2 detail). */
export const BaseStationOperationsPanel: React.FC<
  BaseStationOperationsPanelProps
> = ({
  bsEui,
  online,
  certificateExpiresAt,
  certificateFingerprint,
  lastHandshake,
}) => (
  <Grid container spacing={componentSpacing.cardSection.sectionGap}>
    <Grid size={componentSpacing.gridSpan.fiveTwelfths}>
      <BaseStationAvailabilityCard bsEui={bsEui} />
    </Grid>
    <Grid size={componentSpacing.gridSpan.third}>
      <BaseStationCertificateCard
        bsEui={bsEui}
        expiresAt={certificateExpiresAt}
        fingerprint={certificateFingerprint}
        lastHandshake={lastHandshake}
      />
    </Grid>
    <Grid size={componentSpacing.gridSpan.quarter}>
      <BaseStationPingCard bsEui={bsEui} online={online} />
    </Grid>
  </Grid>
);
