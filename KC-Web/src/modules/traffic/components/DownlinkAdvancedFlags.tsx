/** The optional dlDataQue flags (SCACI §3.10.1), collapsed by default. */

import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Switch,
  Typography,
} from "@mui/material";

import {
  LABEL_DL_ADVANCED_OPTIONS,
  LABEL_DL_EXP_ONLY,
  LABEL_DL_RESPONSE_EXP,
  LABEL_DL_RESPONSE_PRIO,
  LABEL_DL_RX_STAT_QRY,
  LABEL_DL_WIND_REQ,
} from "@constants/messages";
import { ExpandMoreIcon } from "@theme/icons";

import type { DownlinkForm } from "../hooks";
import type { DownlinkFormValues } from "../utils/downlink-form";

const ADVANCED_FLAGS: Array<{
  label: string;
  field: keyof Pick<
    DownlinkFormValues,
    "responseExp" | "responsePrio" | "dlWindReq" | "expOnly" | "dlRxStatQry"
  >;
}> = [
  { label: LABEL_DL_RESPONSE_EXP, field: "responseExp" },
  { label: LABEL_DL_RESPONSE_PRIO, field: "responsePrio" },
  { label: LABEL_DL_WIND_REQ, field: "dlWindReq" },
  { label: LABEL_DL_EXP_ONLY, field: "expOnly" },
  { label: LABEL_DL_RX_STAT_QRY, field: "dlRxStatQry" },
];

export function DownlinkAdvancedFlags({ form }: { form: DownlinkForm }) {
  const { values, setField } = form;
  return (
    <Accordion disableGutters sx={{ "&:before": { display: "none" } }}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Typography variant="body2">{LABEL_DL_ADVANCED_OPTIONS}</Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Box sx={{ display: "flex", gap: 3, flexWrap: "wrap" }}>
          {ADVANCED_FLAGS.map(({ label, field }) => (
            <Box key={label} sx={{ display: "flex", alignItems: "center" }}>
              <Switch
                checked={values[field]}
                onChange={(_, checked) => setField(field, checked)}
                size="small"
              />
              <Typography variant="body2">{label}</Typography>
            </Box>
          ))}
        </Box>
      </AccordionDetails>
    </Accordion>
  );
}
