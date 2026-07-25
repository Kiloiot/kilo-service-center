/**
 * Shared Communication + Advanced MIOTY + Counter settings section
 * used by both AddEndPointDialog and EditEndPointDialog.
 */

import React from "react";

import { Typography } from "@mui/material";
import Grid from "@mui/material/Grid";

import { ENDPOINT_FORM } from "@constants/messages";

import {
  AdvancedMiotySettings,
  CommunicationSettings,
  CounterFields,
} from "./EndpointFormFields";

interface EndpointSettingsSectionProps {
  formData: {
    bidirectional: boolean;
    preAttach: boolean;
    dualChan: boolean;
    repetition: boolean;
    wideCarrOff: boolean;
    longBlkDist: boolean;
    lastPacketCnt: string;
    attachCnt: string;
  };
  handleChange: (
    field: string,
  ) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  errors: Record<string, string>;
  /** Whether counter fields are required (Add dialog = true, Edit = false). */
  counterRequired?: boolean;
}

const EndpointSettingsSection: React.FC<EndpointSettingsSectionProps> = ({
  formData,
  handleChange,
  errors,
  counterRequired,
}) => (
  <>
    {/* Communication Settings */}
    <Grid size={12}>
      <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
        {ENDPOINT_FORM.SECTION_COMMUNICATION}
      </Typography>
    </Grid>

    <CommunicationSettings
      bidirectional={formData.bidirectional}
      preAttach={formData.preAttach}
      onBidirectionalChange={handleChange("bidirectional")}
      onPreAttachChange={handleChange("preAttach")}
    />

    {/* Advanced MIOTY Settings */}
    <Grid size={12}>
      <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
        {ENDPOINT_FORM.SECTION_ADVANCED}
      </Typography>
    </Grid>

    <AdvancedMiotySettings
      dualChan={formData.dualChan}
      repetition={formData.repetition}
      wideCarrOff={formData.wideCarrOff}
      longBlkDist={formData.longBlkDist}
      onDualChanChange={handleChange("dualChan")}
      onRepetitionChange={handleChange("repetition")}
      onWideCarrOffChange={handleChange("wideCarrOff")}
      onLongBlkDistChange={handleChange("longBlkDist")}
    />

    <CounterFields
      lastPacketCnt={formData.lastPacketCnt}
      attachCnt={formData.attachCnt}
      onLastPacketCntChange={handleChange("lastPacketCnt")}
      onAttachCntChange={handleChange("attachCnt")}
      errors={errors}
      required={counterRequired}
    />
  </>
);

export default EndpointSettingsSection;
