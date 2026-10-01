import React, { useState } from "react";

import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Typography,
} from "@mui/material";

import { logger } from "@utils/logger";
import { TIMING_COPY_FEEDBACK } from "@constants/app";
import {
  API_KEY_CREATED_DIALOG,
  API_KEYS_PAGE,
  ERR_COPY_API_KEY,
} from "@constants/messages";
import { ContentCopyIcon } from "@theme/icons";

interface ApiKeyRevealDialogProps {
  rawKey: string;
  onCopied: () => void;
  onClose: () => void;
}

/** Shows a newly created key once, with a copy button. */
const ApiKeyRevealDialog: React.FC<ApiKeyRevealDialogProps> = ({
  rawKey,
  onCopied,
  onClose,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(rawKey);
      setCopied(true);
      onCopied();
      setTimeout(() => setCopied(false), TIMING_COPY_FEEDBACK);
    } catch (err) {
      logger.error(ERR_COPY_API_KEY, err);
    }
  };

  return (
    <Dialog open={!!rawKey} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{API_KEY_CREATED_DIALOG.TITLE}</DialogTitle>
      <DialogContent>
        <Alert severity="warning" sx={{ mb: 2 }}>
          {API_KEY_CREATED_DIALOG.WARNING}
        </Alert>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {API_KEY_CREATED_DIALOG.LABEL_API_KEY}
        </Typography>
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            gap: 1,
            p: 2,
            bgcolor: "grey.100",
            borderRadius: 1,
            wordBreak: "break-all",
          }}
        >
          <Typography
            variant="body2"
            sx={(theme) => ({
              flex: 1,
              fontFamily: theme.typography.monoFontFamily,
            })}
          >
            {rawKey}
          </Typography>
          <IconButton
            onClick={handleCopy}
            color={copied ? "success" : "default"}
          >
            <ContentCopyIcon />
          </IconButton>
        </Box>
        {copied && (
          <Typography variant="body2" color="success.main" sx={{ mt: 1 }}>
            {API_KEYS_PAGE.ACTION_COPIED}
          </Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} variant="contained">
          {API_KEY_CREATED_DIALOG.ACTION_CLOSE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ApiKeyRevealDialog;
