import React, { useState } from "react";

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
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Grid,
  IconButton,
  Typography,
} from "@mui/material";
import { format } from "date-fns";

import {
  CERTIFICATE_INFO,
  CERTIFICATE_LABELS,
  CERTIFICATES_PAGE,
  ERR_CERTIFICATE_GENERIC,
  SERVER_CERTIFICATES,
} from "@constants/messages";
import {
  AddIcon,
  CheckCircleIcon,
  ErrorIcon,
  RefreshIcon,
  WarningIcon,
} from "@theme/icons";

interface CertStatus {
  subject: string;
  issuer: string;
  notBefore: Date;
  notAfter: Date;
  daysUntilExpiry: number;
  isValid: boolean;
}

const Certificates: React.FC = () => {
  const [renewDialogOpen, setRenewDialogOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const { data, isLoading, isError, error, refetch } =
    useServerCertificateStatus();
  const generateMutation = useGenerateServerCertificates();
  const renewMutation = useRenewServerCertificates();

  const actionLoading = generateMutation.isPending || renewMutation.isPending;

  const serverCert = data?.serverCert;
  const caCert = data?.caCert;
  const hasServerCert = !!serverCert;
  const hasCerts = serverCert || caCert;

  const handleGenerate = async () => {
    setActionError(null);
    try {
      await generateMutation.mutateAsync();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : ERR_CERTIFICATE_GENERIC,
      );
    }
  };

  const handleRenew = async () => {
    setRenewDialogOpen(false);
    setActionError(null);
    try {
      await renewMutation.mutateAsync();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : ERR_CERTIFICATE_GENERIC,
      );
    }
  };

  const getStatusChip = (cert: CertStatus) => {
    if (!cert.isValid || cert.daysUntilExpiry < 0) {
      return {
        color: "error" as const,
        text: CERTIFICATES_PAGE.EXPIRED,
        icon: <ErrorIcon />,
      };
    }
    if (cert.daysUntilExpiry <= 30) {
      return {
        color: "warning" as const,
        text: `${CERTIFICATE_INFO.EXPIRES_IN_PREFIX}${cert.daysUntilExpiry}${CERTIFICATE_INFO.EXPIRES_IN_SUFFIX}`,
        icon: <WarningIcon />,
      };
    }
    return {
      color: "success" as const,
      text: CERTIFICATES_PAGE.VALID,
      icon: <CheckCircleIcon />,
    };
  };

  const renderCertCard = (cert: CertStatus, title: string) => {
    const status = getStatusChip(cert);
    return (
      <Grid size={{ xs: 12, md: 6 }} key={title}>
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
              <Chip
                label={status.text}
                color={status.color}
                icon={status.icon}
                size="small"
              />
            </Box>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.ISSUER}: {cert.issuer}
            </Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.SUBJECT}: {cert.subject}
            </Typography>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              {CERTIFICATE_INFO.EXPIRES}: {format(cert.notAfter, "PPP")}
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
          <IconButton onClick={() => refetch()} sx={{ mr: 1 }}>
            <RefreshIcon />
          </IconButton>
          {hasServerCert ? (
            <Button
              variant="contained"
              onClick={() => setRenewDialogOpen(true)}
              disabled={actionLoading}
            >
              {SERVER_CERTIFICATES.RENEW_BUTTON}
            </Button>
          ) : (
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

      <Alert severity="info" sx={{ mb: 3 }}>
        <AlertTitle>{CERTIFICATE_INFO.ALERT_TITLE}</AlertTitle>
        {CERTIFICATE_INFO.ALERT_TEXT}
      </Alert>

      {(actionError || isError) && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {actionError ??
            (error instanceof Error ? error.message : ERR_CERTIFICATE_GENERIC)}
        </Alert>
      )}

      {isLoading ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 5 }}>
          <CircularProgress />
        </Box>
      ) : !hasCerts ? (
        <Box sx={{ display: "flex", justifyContent: "center", py: 5 }}>
          <Typography color="text.secondary">
            {CERTIFICATES_PAGE.NO_CERTIFICATES}
          </Typography>
        </Box>
      ) : (
        <Grid container spacing={3}>
          {serverCert && renderCertCard(serverCert, CERTIFICATE_LABELS.SERVER)}
          {caCert && renderCertCard(caCert, CERTIFICATE_LABELS.CA)}
        </Grid>
      )}

      <Dialog open={renewDialogOpen} onClose={() => setRenewDialogOpen(false)}>
        <DialogTitle>{SERVER_CERTIFICATES.RENEW_CONFIRM_TITLE}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {SERVER_CERTIFICATES.RENEW_CONFIRM_TEXT}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRenewDialogOpen(false)}>
            {SERVER_CERTIFICATES.CANCEL}
          </Button>
          <Button onClick={handleRenew} variant="contained" color="primary">
            {SERVER_CERTIFICATES.RENEW_BUTTON}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Certificates;
