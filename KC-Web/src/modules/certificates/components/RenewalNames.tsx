import React from "react";

import { Box, Chip, Typography } from "@mui/material";

import { SERVER_CERTIFICATES } from "@constants/messages";

interface RenewalNamesProps {
  names: string[];
}

/** What renewing the server certificate does, and the names it will carry. */
const RenewalNames: React.FC<RenewalNamesProps> = ({ names }) => (
  <Box>
    <Typography variant="body2" sx={{ mb: 2 }}>
      {SERVER_CERTIFICATES.RENEW_CONFIRM_TEXT}
    </Typography>
    <Typography variant="subtitle2" gutterBottom>
      {SERVER_CERTIFICATES.RENEW_NAMES_LABEL}
    </Typography>
    <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
      {names.map((name) => (
        <Chip key={name} label={name} size="small" variant="outlined" />
      ))}
    </Box>
  </Box>
);

export default RenewalNames;
