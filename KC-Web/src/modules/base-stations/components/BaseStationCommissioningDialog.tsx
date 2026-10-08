import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Typography,
} from "@mui/material";

import { useClipboard } from "@hooks/useClipboard";
import { COMMISSIONING_PHASE, type CommissioningPhase } from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_CONTINUE,
  ACTION_CREATING,
  ACTION_NEXT,
  ACTION_RETRY_CERTS,
  ACTION_RETRYING_CERTS,
  TITLE_ADD_BS,
  TITLE_BS_PARTIAL,
  TITLE_BS_READY,
} from "@constants/messages";
import { CloseIcon, SecurityIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { useCertificateBundleDownload, useCommissioningWizard } from "../hooks";
import CommissioningInputStep from "./CommissioningInputStep";
import CommissioningPartialStep from "./CommissioningPartialStep";
import CommissioningSuccessStep from "./CommissioningSuccessStep";

interface BaseStationCommissioningDialogProps {
  open: boolean;
  onClose: () => void;
}

const PHASE_TITLES: Record<CommissioningPhase, string> = {
  [COMMISSIONING_PHASE.INPUT]: TITLE_ADD_BS,
  [COMMISSIONING_PHASE.PARTIAL]: TITLE_BS_PARTIAL,
  [COMMISSIONING_PHASE.SUCCESS]: TITLE_BS_READY,
};

export default function BaseStationCommissioningDialog({
  open,
  onClose,
}: BaseStationCommissioningDialogProps) {
  const wizard = useCommissioningWizard();
  const clipboard = useClipboard();
  const handleDownload = useCertificateBundleDownload(
    wizard.certificateData?.downloadUrls.caCert,
    wizard.failWith,
  );

  const handleClose = () => {
    wizard.reset();
    clipboard.reset();
    onClose();
  };

  const pendingIcon = wizard.isPending ? (
    <CircularProgress size={componentSpacing.spinner.button} />
  ) : (
    <SecurityIcon />
  );

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{ sx: { bgcolor: "background.paper" } }}
    >
      <DialogTitle>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box display="flex" alignItems="center" gap={1}>
            <SecurityIcon
              color={
                wizard.phase === COMMISSIONING_PHASE.PARTIAL
                  ? "warning"
                  : "primary"
              }
            />
            <Typography variant="h6">{PHASE_TITLES[wizard.phase]}</Typography>
          </Box>
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>

      <DialogContent>
        {wizard.phase === COMMISSIONING_PHASE.INPUT && (
          <CommissioningInputStep
            values={wizard.values}
            errors={wizard.errors}
            disabled={wizard.isPending}
            onFieldChange={wizard.setField}
            onLocationPick={wizard.setLocation}
          />
        )}
        {wizard.phase === COMMISSIONING_PHASE.PARTIAL && (
          <CommissioningPartialStep
            name={wizard.values.name}
            eui={wizard.values.eui}
            errorMessage={wizard.errors.general}
            clipboard={clipboard}
          />
        )}
        {wizard.phase === COMMISSIONING_PHASE.SUCCESS && (
          <CommissioningSuccessStep
            name={wizard.values.name}
            eui={wizard.values.eui}
            certificateData={wizard.certificateData}
            errorMessage={wizard.errors.general}
            clipboard={clipboard}
            onDownload={handleDownload}
          />
        )}
      </DialogContent>

      <DialogActions>
        {wizard.phase === COMMISSIONING_PHASE.INPUT && (
          <>
            <Button onClick={handleClose} disabled={wizard.isPending}>
              {ACTION_CANCEL}
            </Button>
            <Button
              onClick={wizard.submit}
              variant="contained"
              disabled={
                wizard.isPending ||
                !wizard.values.eui ||
                !wizard.values.name.trim()
              }
              startIcon={pendingIcon}
            >
              {wizard.isPending ? ACTION_CREATING : ACTION_NEXT}
            </Button>
          </>
        )}
        {wizard.phase === COMMISSIONING_PHASE.PARTIAL && (
          <>
            <Button onClick={handleClose} disabled={wizard.isPending}>
              {ACTION_CONTINUE}
            </Button>
            <Button
              onClick={wizard.retryCerts}
              variant="contained"
              color="warning"
              disabled={wizard.isPending}
              startIcon={pendingIcon}
            >
              {wizard.isPending ? ACTION_RETRYING_CERTS : ACTION_RETRY_CERTS}
            </Button>
          </>
        )}
        {wizard.phase === COMMISSIONING_PHASE.SUCCESS && (
          <Button onClick={handleClose} variant="contained">
            {ACTION_CONTINUE}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
