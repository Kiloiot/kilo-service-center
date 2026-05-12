import React, { useEffect, useState } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";

import type { AuthSettingsAPI, LoginRequest } from "@api-types/api";
import { isUnauthorizedError } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  TextField,
  Typography,
} from "@mui/material";

import { apiService } from "@services/api";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useSystem } from "@contexts/SystemContext";
import { persistAuthSession } from "@utils/auth-session";
import { AUTH_LAYOUT, ROUTES } from "@constants/app";
import {
  ACTION_CREATE_ONE,
  ACTION_LOGIN,
  ACTION_LOGIN_PROVIDER,
  ACTION_NO_ACCOUNT,
  ERR_AUTH_INVALID_CREDENTIALS,
  ERR_AUTH_LOGIN_FAILED,
  ERR_AUTH_ORG_REQUIRED,
  ERR_AUTH_SETTINGS_LOAD,
  LABEL_EMAIL,
  LABEL_PASSWORD,
  VAL_EMAIL_REQUIRED,
  VAL_PASSWORD_REQUIRED,
} from "@constants/messages";

import AuthBranding from "../components/AuthBranding";

const Login: React.FC = () => {
  const { versionInfo } = useSystem();
  const [settings, setSettings] = useState<AuthSettingsAPI | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [validation, setValidation] = useState<{
    email?: string;
    password?: string;
  }>({});
  const navigate = useNavigate();
  const { setOrganization } = useOrganization();
  const { setUser } = useSession();

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const authSettings = await apiService.getAuthSettings();
        setSettings(authSettings);

        // Auto-redirect if configured (provider takes precedence)
        if (authSettings.oidc?.enabled && authSettings.oidc.login_redirect) {
          window.location.replace(authSettings.oidc.login_url);
          return;
        }
        if (
          authSettings.oauth2?.enabled &&
          authSettings.oauth2.login_redirect
        ) {
          window.location.replace(authSettings.oauth2.login_url);
          return;
        }
      } catch {
        setError(ERR_AUTH_SETTINGS_LOAD);
      } finally {
        setLoading(false);
      }
    };
    fetchSettings();
  }, []);

  const validateForm = (): boolean => {
    const errors: { email?: string; password?: string } = {};
    if (!email.trim()) errors.email = VAL_EMAIL_REQUIRED;
    if (!password) errors.password = VAL_PASSWORD_REQUIRED;
    setValidation(errors);
    return Object.keys(errors).length === 0;
  };

  const handleLocalLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) return;

    setSubmitting(true);
    setError(null);
    try {
      const payload: LoginRequest = { email: email.trim(), password };
      const loginResponse = await apiService.login(payload);

      const session = persistAuthSession(loginResponse);
      setUser(session.user);
      setOrganization(session.orgId, session.orgName, session.user.id);
      navigate(ROUTES.HOME);
    } catch (err) {
      if (err instanceof Error && err.message === ERR_AUTH_ORG_REQUIRED) {
        setError(ERR_AUTH_ORG_REQUIRED);
      } else if (isUnauthorizedError(err)) {
        setError(ERR_AUTH_INVALID_CREDENTIALS);
      } else {
        setError(ERR_AUTH_LOGIN_FAILED);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleProviderLogin = (loginUrl: string) => {
    window.location.replace(loginUrl);
  };

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight={AUTH_LAYOUT.FULL_HEIGHT}
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box
      display="flex"
      justifyContent="center"
      alignItems="center"
      minHeight={AUTH_LAYOUT.FULL_HEIGHT}
    >
      <Paper
        sx={{
          p: AUTH_LAYOUT.CARD_PADDING,
          maxWidth: AUTH_LAYOUT.CARD_MAX_WIDTH,
          width: "100%",
          textAlign: "center",
        }}
      >
        <AuthBranding versionInfo={versionInfo} />

        {error && (
          <Alert severity="error" sx={{ mb: AUTH_LAYOUT.SPACING_MB }}>
            {error}
          </Alert>
        )}

        {/* Local login form */}
        {settings?.local_login_enabled && (
          <form onSubmit={handleLocalLogin}>
            <TextField
              fullWidth
              label={LABEL_EMAIL}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              error={!!validation.email}
              helperText={validation.email}
              margin="normal"
            />
            <TextField
              fullWidth
              label={LABEL_PASSWORD}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              error={!!validation.password}
              helperText={validation.password}
              margin="normal"
            />
            <Button
              fullWidth
              type="submit"
              variant="contained"
              disabled={submitting}
              sx={{ mt: AUTH_LAYOUT.SPACING_MT }}
            >
              {submitting ? (
                <CircularProgress size={AUTH_LAYOUT.SPINNER_SIZE} />
              ) : (
                ACTION_LOGIN
              )}
            </Button>
            {settings?.registration_enabled && (
              <Typography variant="body2" textAlign="center" sx={{ mt: 1 }}>
                {ACTION_NO_ACCOUNT}{" "}
                <RouterLink
                  to={ROUTES.REGISTER}
                  style={{ textDecoration: "none" }}
                >
                  {ACTION_CREATE_ONE}
                </RouterLink>
              </Typography>
            )}
          </form>
        )}

        {/* OIDC provider button */}
        {settings?.oidc?.enabled && (
          <Button
            fullWidth
            variant="outlined"
            onClick={() => handleProviderLogin(settings.oidc!.login_url)}
            sx={{ mt: AUTH_LAYOUT.SPACING_MT }}
          >
            {`${ACTION_LOGIN_PROVIDER} ${settings.oidc.login_label}`}
          </Button>
        )}

        {/* OAuth2 provider button */}
        {settings?.oauth2?.enabled && (
          <Button
            fullWidth
            variant="outlined"
            onClick={() => handleProviderLogin(settings.oauth2!.login_url)}
            sx={{ mt: AUTH_LAYOUT.SPACING_MT }}
          >
            {`${ACTION_LOGIN_PROVIDER} ${settings.oauth2.login_label}`}
          </Button>
        )}
      </Paper>
    </Box>
  );
};

export default Login;
