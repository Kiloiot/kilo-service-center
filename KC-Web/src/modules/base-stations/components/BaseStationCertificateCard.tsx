import React from "react";

import { Box, Button, Chip, Stack, Typography } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { InfoGrid } from "@components/common/InfoGrid";
import { useFeedback } from "@contexts/feedback";
import { useBaseStationCertificateDownload } from "@hooks/useBaseStationOperations";
import {
  type CertificateExpiry,
  formatCertificateExpiresAt,
  stationCertificateExpiry,
} from "@utils/certificate-expiry";
import { formatDateTime } from "@utils/date-format";
import {
  formatCertificateExpiryState,
  formatDaysLeft,
  formatOptional,
} from "@utils/formatters";
import {
  CERTIFICATE_DOWNLOAD_TYPES,
  CERTIFICATE_EXPIRY_STATE,
  type PublicCertificateType,
} from "@constants/app";
import { BS_OPERATIONS } from "@constants/messages";
import { DownloadIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { certificateDownloadFailure } from "../utils/certificate-download";

interface BaseStationCertificateCardProps {
  bsEui: string;
  expiresAt?: string;
  fingerprint?: string;
  lastHandshake?: string;
}

const DOWNLOADS: ReadonlyArray<{ type: PublicCertificateType; label: string }> =
  [
    { type: CERTIFICATE_DOWNLOAD_TYPES.CA, label: BS_OPERATIONS.DOWNLOAD_CA },
    {
      type: CERTIFICATE_DOWNLOAD_TYPES.CLIENT,
      label: BS_OPERATIONS.DOWNLOAD_CLIENT,
    },
  ];

/** The certificate's state, with the days left once its expiry is known. */
const ExpiryChip: React.FC<{ expiry: CertificateExpiry }> = ({ expiry }) => {
  const state = formatCertificateExpiryState(expiry.state);
  const label =
    expiry.daysUntilExpiry === null
      ? state.label
      : `${state.label} (${formatDaysLeft(expiry.daysUntilExpiry)})`;
  return <Chip size="small" color={state.color} label={label} />;
};

/** The public certificate downloads; a failed one is reported as an error. */
const CertificateDownloads: React.FC<{ bsEui: string }> = ({ bsEui }) => {
  const download = useBaseStationCertificateDownload();
  const feedback = useFeedback();
  return (
    <Box
      sx={{
        display: "flex",
        flexWrap: "wrap",
        gap: componentSpacing.dataCard.chipGap,
      }}
    >
      {DOWNLOADS.map((item) => (
        <Button
          key={item.type}
          size="small"
          variant="outlined"
          startIcon={<DownloadIcon />}
          disabled={download.isPending}
          onClick={() =>
            download.mutate(
              { bsEui, certType: item.type },
              {
                onError: (error) =>
                  feedback.error(certificateDownloadFailure(error)),
              },
            )
          }
        >
          {item.label}
        </Button>
      ))}
    </Box>
  );
};

/**
 * The client certificate a base station is bound to (stationCertificateExpiry
 * states it) and its public downloads: the service center CA and the stored
 * client certificate.
 */
export const BaseStationCertificateCard: React.FC<
  BaseStationCertificateCardProps
> = ({ bsEui, expiresAt, fingerprint, lastHandshake }) => {
  const expiry = stationCertificateExpiry(fingerprint, expiresAt);
  return (
    <DataCard title={BS_OPERATIONS.CERTIFICATE_TITLE}>
      {expiry.state === CERTIFICATE_EXPIRY_STATE.NOT_ISSUED ? (
        <Typography variant="body2" color="text.secondary">
          {BS_OPERATIONS.NOT_ISSUED}
        </Typography>
      ) : (
        <Stack spacing={componentSpacing.infoGrid.rowGap}>
          <InfoGrid
            items={[
              {
                label: BS_OPERATIONS.STATUS,
                value: <ExpiryChip expiry={expiry} />,
              },
              {
                label: BS_OPERATIONS.EXPIRES,
                value: formatCertificateExpiresAt(expiry),
              },
              {
                label: BS_OPERATIONS.FINGERPRINT,
                value: formatOptional(fingerprint),
              },
              {
                label: BS_OPERATIONS.LAST_HANDSHAKE,
                value: formatDateTime(lastHandshake),
              },
            ]}
          />
          <CertificateDownloads bsEui={bsEui} />
        </Stack>
      )}
    </DataCard>
  );
};
