/**
 * Expanded uplink row: every base station reception with its subpackets,
 * the ulData message as the Application Center receives it, the payload in
 * hex, and the blueprint decode of the payload with its status.
 */

import type { ReactNode } from "react";

import type { BaseStationReceptionAPI, UplinkUI } from "@api-types/api";
import { Box, Typography } from "@mui/material";

import { DataTable } from "@components/common/DataTable";
import { HexText } from "@components/common/HexText";
import { JsonPreview } from "@components/common/JsonPreview";
import { formatUserData } from "@utils/formatters";
import { subpacketRows, toUlDataMessage } from "@utils/ul-data";
import { withDecodeErrorCode } from "@utils/uplink-summary";
import { DATA_TABLE_LAYOUT, DECODE_STATUS } from "@constants/app";
import { SCACI_FIELD, UPLINK_TABLE } from "@constants/messages";

import { RECEPTION_COLUMNS, SUBPACKET_COLUMNS } from "./receptionColumns";

function Subpackets({ reception }: { reception: BaseStationReceptionAPI }) {
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {SCACI_FIELD.SUBPACKETS}
      </Typography>
      <DataTable
        columns={SUBPACKET_COLUMNS}
        rows={subpacketRows(reception)}
        rowKey={(row) => String(row.index)}
        emptyMessage={UPLINK_TABLE.NO_SUBPACKETS}
      />
    </Box>
  );
}

function LabeledValue({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {label}
      </Typography>
      {children}
    </Box>
  );
}

// Only a decode that ran has a result to show; skipped and pending say nothing.
const DECODE_RAN = new Set<string>([
  DECODE_STATUS.SUCCESS,
  DECODE_STATUS.FAILED,
]);

function DecodeResult({ uplink }: { uplink: UplinkUI }) {
  if (!DECODE_RAN.has(uplink.decodeStatus)) return null;
  return (
    <>
      <LabeledValue label={UPLINK_TABLE.DECODE_STATUS}>
        <Typography variant="body2">
          {withDecodeErrorCode(uplink.decodeStatus, uplink.decodeErrorCode)}
        </Typography>
      </LabeledValue>
      {uplink.decodedPayload && (
        <JsonPreview
          title={UPLINK_TABLE.DECODED_PAYLOAD}
          value={uplink.decodedPayload}
        />
      )}
    </>
  );
}

export function UplinkDetails({ uplink }: { uplink: UplinkUI }) {
  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        gap: DATA_TABLE_LAYOUT.SECTION_GAP,
      }}
    >
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          {SCACI_FIELD.BASE_STATIONS}
        </Typography>
        <DataTable
          columns={RECEPTION_COLUMNS}
          rows={uplink.receptions}
          rowKey={(reception) => reception.bsEui}
          emptyMessage={UPLINK_TABLE.NO_RECEPTIONS}
          renderDetails={(reception) => <Subpackets reception={reception} />}
        />
      </Box>
      <JsonPreview
        title={SCACI_FIELD.UL_DATA}
        value={toUlDataMessage(uplink)}
      />
      <LabeledValue label={UPLINK_TABLE.PAYLOAD_HEX}>
        <HexText>{formatUserData(uplink.userData)}</HexText>
      </LabeledValue>
      <DecodeResult uplink={uplink} />
    </Box>
  );
}
