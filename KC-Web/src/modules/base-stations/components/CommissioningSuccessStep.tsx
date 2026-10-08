import {
  Alert,
  Box,
  Divider,
  IconButton,
  Tooltip,
  Typography,
} from "@mui/material";

import type { ClipboardControls } from "@hooks/useClipboard";
import { formatEui } from "@utils/eui";
import { getMonoBody2 } from "@utils/typography";
import type { CertificateDownloadType } from "@constants/app";
import { BS_COPY_FIELDS } from "@constants/app";
import {
  ACTION_COPIED,
  ACTION_COPY,
  DATA_TABLE,
  INSTR_BS_CREATED_FALLBACK,
  INSTR_STEP_1,
  INSTR_STEP_2_HEADER,
  INSTR_STEP_3,
  INSTR_TLS_NOTE,
  LABEL_CA_CERT_FILE,
  LABEL_CLIENT_CERT_PREFIX,
  LABEL_CLIENT_CERT_SUFFIX,
  LABEL_PRIVATE_KEY_PREFIX,
  LABEL_PRIVATE_KEY_SUFFIX,
  LABEL_SC_URL_DISPLAY,
  MSG_BS_ADDED_NO_CERTS,
  MSG_BS_ADDED_WITH_CERTS,
  SECTION_NEXT_STEPS,
} from "@constants/messages";
import { CheckCircleIcon, ContentCopyIcon } from "@theme/icons";

import type { CertificateData } from "../utils/commissioning-form";
import BsConfigSummary from "./BsConfigSummary";
import CertificateDownloadButtons from "./CertificateDownloadButtons";
import ServiceCenterUrlWarning from "./ServiceCenterUrlWarning";

interface CommissioningSuccessStepProps {
  name: string;
  eui: string;
  certificateData: CertificateData | null;
  errorMessage?: string;
  clipboard: ClipboardControls;
  onDownload: (certType: CertificateDownloadType) => void;
}

export default function CommissioningSuccessStep({
  name,
  eui,
  certificateData,
  errorMessage,
  clipboard,
  onDownload,
}: CommissioningSuccessStepProps) {
  const serviceCenterUrl = certificateData?.serviceCenterUrl;
  return (
    <Box sx={{ pt: 2 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        {certificateData ? MSG_BS_ADDED_WITH_CERTS : MSG_BS_ADDED_NO_CERTS}
      </Alert>

      <BsConfigSummary
        name={name}
        eui={eui}
        copiedField={clipboard.copiedField}
        onCopy={clipboard.copy}
        monoSxGetter={getMonoBody2}
      >
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Typography variant="body2" color="text.secondary">
            {LABEL_SC_URL_DISPLAY}
          </Typography>
          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
              {serviceCenterUrl || DATA_TABLE.NO_VALUE}
            </Typography>
            {serviceCenterUrl && (
              <Tooltip
                title={
                  clipboard.copiedField === BS_COPY_FIELDS.SC_URL
                    ? ACTION_COPIED
                    : ACTION_COPY
                }
              >
                <IconButton
                  size="small"
                  onClick={() =>
                    clipboard.copy(serviceCenterUrl, BS_COPY_FIELDS.SC_URL)
                  }
                >
                  {clipboard.copiedField === BS_COPY_FIELDS.SC_URL ? (
                    <CheckCircleIcon fontSize="small" color="success" />
                  ) : (
                    <ContentCopyIcon fontSize="small" />
                  )}
                </IconButton>
              </Tooltip>
            )}
          </Box>
        </Box>
      </BsConfigSummary>

      {certificateData && !serviceCenterUrl && <ServiceCenterUrlWarning />}

      {certificateData && (
        <CertificateDownloadButtons onDownload={onDownload} />
      )}

      {errorMessage && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {errorMessage}
        </Alert>
      )}

      <Divider sx={{ my: 3 }} />

      <Typography variant="subtitle2" gutterBottom>
        {SECTION_NEXT_STEPS}
      </Typography>
      <Box sx={{ mt: 1 }}>
        {certificateData ? (
          <>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_1}
            </Typography>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_2_HEADER}
              <br />
              &nbsp;&nbsp;• {LABEL_SC_URL_DISPLAY}{" "}
              {certificateData.serviceCenterUrl}
              <br />
              &nbsp;&nbsp;• {LABEL_CA_CERT_FILE}
              <br />
              &nbsp;&nbsp;• {LABEL_CLIENT_CERT_PREFIX}
              {formatEui(certificateData.bsEui)}
              {LABEL_CLIENT_CERT_SUFFIX}
              <br />
              &nbsp;&nbsp;• {LABEL_PRIVATE_KEY_PREFIX}
              {formatEui(certificateData.bsEui)}
              {LABEL_PRIVATE_KEY_SUFFIX}
            </Typography>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_3}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {INSTR_TLS_NOTE}
            </Typography>
          </>
        ) : (
          <Typography variant="body2" color="text.secondary">
            {INSTR_BS_CREATED_FALLBACK}
          </Typography>
        )}
      </Box>
    </Box>
  );
}
