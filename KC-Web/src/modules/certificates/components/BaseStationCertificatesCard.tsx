import React from "react";
import { useNavigate } from "react-router-dom";

import { Chip, IconButton, Tooltip } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { DataTable, type DataTableColumn } from "@components/common/DataTable";
import { useBaseStationCertificates } from "@hooks/useBaseStationOperations";
import {
  type BaseStationCertificate,
  formatCertificateExpiresAt,
} from "@utils/certificate-expiry";
import { formatDateTime } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import {
  formatCertificateExpiryState,
  formatDaysLeft,
  formatOptional,
} from "@utils/formatters";
import { BS_CERTIFICATES_SECTION } from "@constants/messages";
import { baseStationDetailPath } from "@router/paths";
import { OpenInNewIcon } from "@theme/icons";

/** NAV 6 BS certificate list: every base station with its issued certificate's expiry. */
export const BaseStationCertificatesCard: React.FC = () => {
  const navigate = useNavigate();
  const { data, isLoading, error } = useBaseStationCertificates();

  const openStation = (certificate: BaseStationCertificate) =>
    navigate(baseStationDetailPath(certificate.eui));

  const columns: DataTableColumn<BaseStationCertificate>[] = [
    {
      id: "name",
      header: BS_CERTIFICATES_SECTION.COL_NAME,
      render: (c) => formatOptional(c.name),
    },
    {
      id: "eui",
      header: BS_CERTIFICATES_SECTION.COL_BS_EUI,
      render: (c) => formatEui(c.eui),
    },
    {
      id: "status",
      header: BS_CERTIFICATES_SECTION.COL_STATUS,
      render: (c) => {
        const state = formatCertificateExpiryState(c.state);
        return <Chip size="small" label={state.label} color={state.color} />;
      },
    },
    {
      id: "expires",
      header: BS_CERTIFICATES_SECTION.COL_EXPIRES,
      render: formatCertificateExpiresAt,
    },
    {
      id: "daysLeft",
      header: BS_CERTIFICATES_SECTION.COL_DAYS_LEFT,
      render: (c) => formatDaysLeft(c.daysUntilExpiry),
    },
    {
      id: "fingerprint",
      header: BS_CERTIFICATES_SECTION.COL_FINGERPRINT,
      render: (c) => formatOptional(c.fingerprint),
    },
    {
      id: "lastHandshake",
      header: BS_CERTIFICATES_SECTION.COL_LAST_HANDSHAKE,
      render: (c) => formatDateTime(c.lastHandshake),
    },
    {
      id: "actions",
      header: BS_CERTIFICATES_SECTION.COL_ACTIONS,
      align: "right",
      render: (c) => (
        <Tooltip title={BS_CERTIFICATES_SECTION.ACTION_OPEN}>
          <IconButton
            size="small"
            aria-label={BS_CERTIFICATES_SECTION.ACTION_OPEN}
            onClick={() => openStation(c)}
          >
            <OpenInNewIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      ),
    },
  ];

  return (
    <DataCard
      title={BS_CERTIFICATES_SECTION.TITLE}
      subtitle={BS_CERTIFICATES_SECTION.SUBTITLE}
      isLoading={isLoading}
      error={error}
      errorFallback={BS_CERTIFICATES_SECTION.ERR_LOAD}
    >
      <DataTable
        columns={columns}
        rows={data ?? []}
        rowKey={(c) => c.eui}
        emptyMessage={BS_CERTIFICATES_SECTION.EMPTY}
      />
    </DataCard>
  );
};
