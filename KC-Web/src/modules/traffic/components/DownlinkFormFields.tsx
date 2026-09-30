/**
 * dlDataQue fields (SCACI §3.10.1): payload (single or counter-dependent
 * rows), format, priority and the protocol flags. Shared by composing a new
 * downlink and editing a pending one.
 */

import { Box, Switch, TextField, Tooltip, Typography } from "@mui/material";

import { DOWNLINK_LIMITS, DOWNLINK_PRIORITY_PRESETS } from "@constants/app";
import {
  DOWNLINK_PRIORITY_LABELS,
  HELPER_DL_CNT_DEPEND,
  HELPER_DL_FORMAT,
  LABEL_DL_CNT_DEPEND,
  LABEL_DL_FORMAT,
  LABEL_DL_PRIORITY,
} from "@constants/messages";
import { componentSpacing } from "@theme/index";

import type { DownlinkForm } from "../hooks";
import { DownlinkAdvancedFlags } from "./DownlinkAdvancedFlags";
import { DownlinkCounterRows } from "./DownlinkCounterRows";
import { DownlinkPayloadInput } from "./DownlinkPayloadInput";

const PRIORITY_PRESETS_HINT = DOWNLINK_PRIORITY_PRESETS.map(
  (p) => `${p.value}=${DOWNLINK_PRIORITY_LABELS[p.preset]}`,
).join(", ");

function DownlinkNumberFields({ form }: { form: DownlinkForm }) {
  const { values, setField, validation } = form;
  return (
    <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
      <TextField
        label={LABEL_DL_FORMAT}
        value={values.format}
        onChange={(e) => setField("format", e.target.value)}
        error={!!validation.formatError}
        helperText={validation.formatError || HELPER_DL_FORMAT}
        type="number"
        size="small"
        sx={{ width: componentSpacing.downlinkForm.formatWidth }}
      />
      <TextField
        label={LABEL_DL_PRIORITY}
        value={values.priority}
        onChange={(e) => setField("priority", e.target.value)}
        error={!!validation.priorityError}
        helperText={validation.priorityError || PRIORITY_PRESETS_HINT}
        type="number"
        inputProps={{ step: DOWNLINK_LIMITS.PRIORITY_STEP }}
        size="small"
        sx={{ width: componentSpacing.downlinkForm.priorityWidth }}
      />
    </Box>
  );
}

export function DownlinkFormFields({ form }: { form: DownlinkForm }) {
  const { values, setField } = form;
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
      {values.cntDepend ? (
        <DownlinkCounterRows form={form} />
      ) : (
        <DownlinkPayloadInput form={form} />
      )}
      <DownlinkNumberFields form={form} />
      <Box sx={{ display: "flex", alignItems: "center" }}>
        <Switch
          checked={values.cntDepend}
          onChange={(_, checked) => setField("cntDepend", checked)}
          size="small"
        />
        <Tooltip title={HELPER_DL_CNT_DEPEND}>
          <Typography variant="body2">{LABEL_DL_CNT_DEPEND}</Typography>
        </Tooltip>
      </Box>
      <DownlinkAdvancedFlags form={form} />
    </Box>
  );
}
