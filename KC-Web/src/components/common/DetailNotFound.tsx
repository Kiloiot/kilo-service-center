/** A detail page whose record is missing or failed to load: the reason and the way back to its list. */

import { Alert, Box } from "@mui/material";

import { BackButton } from "./BackButton";

interface DetailNotFoundProps {
  message: string;
  backLabel: string;
  onBack: () => void;
}

export function DetailNotFound({
  message,
  backLabel,
  onBack,
}: DetailNotFoundProps) {
  return (
    <Box sx={{ p: 3, pt: 4 }}>
      <Alert severity="error">{message}</Alert>
      <BackButton label={backLabel} onClick={onBack} sx={{ mt: 2 }} />
    </Box>
  );
}
