import React, { useEffect, useState } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";

import type { AuthSettingsAPI, LoginRequest } from "@api-types/api";
import { ApiError } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Link as MuiLink,
  Paper,
  TextField,
  Typography,
} from "@mui/material";

import { apiService } from "@services/api";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useSystem } from "@contexts/SystemContext";
import { storageService } from "@utils/storage";
import {
  APP_EDITION,
  APP_NAME,
  AUTH_LAYOUT,
  DEFAULT_ORG_NAME,
  ROUTES,
  STORAGE_KEYS,
} from "@constants/app";
import {
  ACTION_CREATE_ONE,
  ACTION_LOGIN,
  ACTION_LOGIN_PROVIDER,
  ACTION_NO_ACCOUNT,
  BRAND,
  ERR_AUTH_INVALID_CREDENTIALS,
  ERR_AUTH_LOGIN_FAILED,
  ERR_AUTH_ORG_REQUIRED,
  ERR_AUTH_SETTINGS_LOAD,
  LABEL_EMAIL,
  LABEL_PASSWORD,
  VAL_EMAIL_REQUIRED,
  VAL_PASSWORD_REQUIRED,
} from "@constants/messages";
import kiloLogo from "@assets/kilo-logo.png";

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

      // Store access token for subsequent API requests
      storageService.setItem(
        STORAGE_KEYS.AUTH_TOKEN,
        loginResponse.tokens.accessToken,
      );

      // Store refresh token if provided (refresh_token_enabled=true on backend)
      if (loginResponse.tokens.refreshToken) {
        storageService.setItem(
          STORAGE_KEYS.REFRESH_TOKEN,
          loginResponse.tokens.refreshToken,
        );
      } else {
        storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
      }

      // Set organization context from user profile
      const { user } = loginResponse;
      const defaultOrg = user.memberships.find(
        (m) => m.orgId === user.defaultOrgId,
      );
      const firstOrg = user.memberships[0];
      const org = defaultOrg || firstOrg;

      if (!org) {
        storageService.removeItem(STORAGE_KEYS.AUTH_TOKEN);
        setError(ERR_AUTH_ORG_REQUIRED);
        return;
      }

      setUser(user);

      setOrganization(org.orgId, org.orgName || DEFAULT_ORG_NAME, user.id);
      navigate(ROUTES.HOME);
    } catch (err) {
      // Distinguish 401 (invalid credentials) from other errors
      if (err instanceof ApiError && err.status === 401) {
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
        {/* Logo and branding */}
        <Box
          sx={{
            mb: 3,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <img
            src={kiloLogo}
            alt={APP_NAME}
            style={{ maxWidth: "200px", height: "auto", marginBottom: "8px" }}
          />
          <Chip
            label={versionInfo?.edition ?? APP_EDITION}
            size="small"
            variant="outlined"
            sx={{ mb: 2 }}
          />
          <Box
            sx={{ display: "flex", gap: 2, justifyContent: "center", mb: 3 }}
          >
            <MuiLink
              href={versionInfo?.documentationUrl}
              target="_blank"
              rel="noopener"
              variant="caption"
            >
              {BRAND.DOCUMENTATION}
            </MuiLink>
            <MuiLink
              href={versionInfo?.sourceUrl}
              target="_blank"
              rel="noopener"
              variant="caption"
            >
              {BRAND.SOURCE}
            </MuiLink>
            <MuiLink
              href={versionInfo?.licenseUrl}
              target="_blank"
              rel="noopener"
              variant="caption"
            >
              {BRAND.LICENSE}
            </MuiLink>
          </Box>
        </Box>

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
