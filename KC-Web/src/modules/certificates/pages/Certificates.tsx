import React, { useState } from "react";

import type { CertificateSummary } from "@api-types/api";
import {
  useGenerateServerCertificates,
  useRenewServerCertificates,
  useServerCertificateStatus,
} from "@hooks";
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  Typography,
} from "@mui/material";
import { ConfirmDialog } from "@ui";

import { useFeedback } from "@contexts/feedback";
import { useCapabilities } from "@hooks/useCapabilities";
import { certificateExpiryState } from "@utils/certificate-expiry";
import { formatDate } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import { formatCertificateExpiryState } from "@utils/formatters";
import { CERTIFICATE_EXPIRY_STATE } from "@constants/app";
import {
  CERTIFICATE_INFO,
  CERTIFICATE_LABELS,
  CERTIFICATES_PAGE,
  ERR_CERTIFICATE_GENERIC,
  SERVER_CERTIFICATES,
} from "@constants/messages";
import { AddIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { BaseStationCertificatesCard } from "../components/BaseStationCertificatesCard";
import RenewalNames from "../components/RenewalNames";

const SECTION_GAP = componentSpacing.cardSection.sectionGap;

const Certificates: React.FC = () => {
  // Server certificates belong to the whole installation: administrators generate and renew them
  const { isServerAdmin } = useCapabilities();
  const [renewDialogOpen, setRenewDialogOpen] = useState(false);
  const feedback = useFeedback();

  const { data, isLoading, isError, error } = useServerCertificateStatus();
  const generateMutation = useGenerateServerCertificates();
  const renewMutation = useRenewServerCertificates();

  const actionLoading = generateMutation.isPending || renewMutation.isPending;

  const serverCert = data?.serverCert;
  const caCert = data?.caCert;
  const hasServerCert = !!serverCert;
  const hasCerts = serverCert || caCert;

  const handleGenerate = async () => {
    try {
      await generateMutation.mutateAsync();
      feedback.success(SERVER_CERTIFICATES.MSG_GENERATED);
    } catch (err) {
      feedback.error(getErrorMessage(err, ERR_CERTIFICATE_GENERIC));
    }
  };

  const handleRenew = async () => {
    await renewMutation.mutateAsync();
    setRenewDialogOpen(false);
    feedback.success(SERVER_CERTIFICATES.MSG_RENEWED);
  };

  const getStatusChip = (cert: CertificateSummary) =>
    formatCertificateExpiryState(
      cert.isValid
        ? certificateExpiryState(cert.daysUntilExpiry)
        : CERTIFICATE_EXPIRY_STATE.EXPIRED,
    );

  const renderCertCard = (cert: CertificateSummary, title: string) => {
    const status = getStatusChip(cert);
    return (
      <Grid size={componentSpacing.gridSpan.half} key={title}>
        <Card>
          <CardContent>
            <Box
              sx={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "flex-start",
                mb: 2,
              }}
            >
              <Typography variant="h6" component="div">
                {title}
              </Typography>
              <Chip label={status.label} color={status.color} size="small" />
            </Box>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.ISSUER}: {cert.issuer}
            </Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.SUBJECT}: {cert.subject}
            </Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.EXPIRES}: {formatDate(cert.notAfter)}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {CERTIFICATE_INFO.DAYS_UNTIL_EXPIRY}: {cert.daysUntilExpiry}
            </Typography>
          </CardContent>
        </Card>
      </Grid>
    );
  };

  return (
    <Box data-testid="certificates-page" sx={{ p: 3, pt: 4 }}>
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          mb: 3,
        }}
      >
        <Typography variant="h4">{CERTIFICATES_PAGE.TITLE}</Typography>
        <Box>
          {isServerAdmin && hasServerCert && (
            <Button
              variant="contained"
              onClick={() => setRenewDialogOpen(true)}
              disabled={actionLoading}
            >
              {SERVER_CERTIFICATES.RENEW_BUTTON}
            </Button>
          )}
          {isServerAdmin && !hasServerCert && (
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleGenerate}
              disabled={actionLoading}
            >
              {SERVER_CERTIFICATES.GENERATE_BUTTON}
            </Button>
          )}
        </Box>
      </Box>

      <Alert severity="info" sx={{ mb: SECTION_GAP }}>
        <AlertTitle>{CERTIFICATE_INFO.ALERT_TITLE}</AlertTitle>
        {CERTIFICATE_INFO.ALERT_TEXT}
      </Alert>

      {isError && (
        <Alert severity="error" sx={{ mb: SECTION_GAP }}>
          {getErrorMessage(error, ERR_CERTIFICATE_GENERIC)}
        </Alert>
      )}

      <Grid container spacing={SECTION_GAP}>
        {isLoading ? (
          <Grid size={componentSpacing.gridSpan.full}>
            <Box sx={{ display: "flex", justifyContent: "center", py: 5 }}>
              <CircularProgress />
            </Box>
          </Grid>
        ) : !hasCerts ? (
          <Grid size={componentSpacing.gridSpan.full}>
            <Box sx={{ display: "flex", justifyContent: "center", py: 5 }}>
              <Typography color="text.secondary">
                {CERTIFICATES_PAGE.NO_CERTIFICATES}
              </Typography>
            </Box>
          </Grid>
        ) : (
          <>
            {serverCert &&
              renderCertCard(serverCert, CERTIFICATE_LABELS.SERVER)}
            {caCert && renderCertCard(caCert, CERTIFICATE_LABELS.CA)}
          </>
        )}
        <Grid size={componentSpacing.gridSpan.full}>
          <BaseStationCertificatesCard />
        </Grid>
      </Grid>

      <ConfirmDialog
        open={renewDialogOpen}
        onClose={() => setRenewDialogOpen(false)}
        onConfirm={handleRenew}
        errorFallback={ERR_CERTIFICATE_GENERIC}
        color="primary"
        title={SERVER_CERTIFICATES.RENEW_CONFIRM_TITLE}
        message={<RenewalNames names={data?.renewalNames ?? []} />}
        confirmLabel={SERVER_CERTIFICATES.RENEW_BUTTON}
        cancelLabel={SERVER_CERTIFICATES.CANCEL}
      />
    </Box>
  );
};

export default Certificates;
