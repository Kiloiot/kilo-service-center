import { Alert, Box } from "@mui/material";

import type { ClipboardControls } from "@hooks/useClipboard";
import { getMonoBody2 } from "@utils/typography";
import { MSG_BS_PARTIAL_SUCCESS } from "@constants/messages";

import BsConfigSummary from "./BsConfigSummary";

interface CommissioningPartialStepProps {
  name: string;
  eui: string;
  errorMessage?: string;
  clipboard: ClipboardControls;
}

export default function CommissioningPartialStep({
  name,
  eui,
  errorMessage,
  clipboard,
}: CommissioningPartialStepProps) {
  return (
    <Box sx={{ pt: 2 }}>
      <Alert severity="warning" sx={{ mb: 3 }}>
        {MSG_BS_PARTIAL_SUCCESS}
      </Alert>
      <BsConfigSummary
        name={name}
        eui={eui}
        copiedField={clipboard.copiedField}
        onCopy={clipboard.copy}
        monoSxGetter={getMonoBody2}
      />
      {errorMessage && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {errorMessage}
        </Alert>
      )}
    </Box>
  );
}
