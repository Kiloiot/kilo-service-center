import React from "react";

import { Box, Button, Typography } from "@mui/material";

import { API_KEYS_PAGE } from "@constants/messages";
import { AddIcon } from "@theme/icons";

/** Page title with the create action. */
const ApiKeysHeader: React.FC<{
  onCreate: () => void;
}> = ({ onCreate }) => (
  <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        {API_KEYS_PAGE.TITLE}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {API_KEYS_PAGE.DESCRIPTION}
      </Typography>
    </Box>
    <Button variant="contained" startIcon={<AddIcon />} onClick={onCreate}>
      {API_KEYS_PAGE.ACTION_CREATE}
    </Button>
  </Box>
);

export default ApiKeysHeader;
