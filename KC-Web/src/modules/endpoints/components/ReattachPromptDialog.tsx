/**
 * Offered after an edit changed the station profile the base stations hold:
 * re-attach now so they pick the new profile up, or later.
 */

import { useId } from "react";

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  Typography,
} from "@mui/material";

import { ENDPOINT_FORM } from "@constants/messages";

interface ReattachPromptDialogProps {
  open: boolean;
  pending: boolean;
  changedFields: string[];
  onReattach: () => void;
  onLater: () => void;
}

export function ReattachPromptDialog({
  open,
  pending,
  changedFields,
  onReattach,
  onLater,
}: ReattachPromptDialogProps) {
  const titleId = useId();
  return (
    <Dialog open={open} onClose={onLater} aria-labelledby={titleId}>
      <DialogTitle id={titleId}>
        {ENDPOINT_FORM.REATTACH_DIALOG_TITLE}
      </DialogTitle>
      <DialogContent>
        <Alert severity="info" sx={{ mt: 1 }}>
          {ENDPOINT_FORM.ALERT_REATTACH_PROMPT}
        </Alert>
        <Typography variant="body2" sx={{ mt: 2 }}>
          {ENDPOINT_FORM.REATTACH_CHANGED_FIELDS}
        </Typography>
        <List dense disablePadding>
          {changedFields.map((label) => (
            <ListItem key={label} sx={{ display: "list-item", ml: 3 }}>
              {label}
            </ListItem>
          ))}
        </List>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {ENDPOINT_FORM.REATTACH_LATER_HINT}
        </Typography>
      </DialogContent>
      <DialogActions>
        <Button onClick={onLater}>{ENDPOINT_FORM.REATTACH_BUTTON_LATER}</Button>
        <Button onClick={onReattach} variant="contained" disabled={pending}>
          {pending
            ? ENDPOINT_FORM.REATTACH_BUTTON_PENDING
            : ENDPOINT_FORM.REATTACH_BUTTON_NOW}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
