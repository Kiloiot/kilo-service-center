import React, { useEffect, useState } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';

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
} from '@mui/material';

import { apiService } from '@services/api';
import { useFeatureFlags } from '@contexts/FeatureFlagContext';
import { useOrganization } from '@contexts/OrganizationContext';
import { useSession } from '@contexts/SessionContext';
import { useSystem } from '@contexts/SystemContext';
import { storageService } from '@utils/storage';
import {
  APP_EDITION,
  APP_NAME,
  AUTH_LAYOUT,
  DEFAULT_ORG_NAME,
  ROUTES,
  STORAGE_KEYS,
} from '@constants/app';
import {
  ACTION_CREATE_ACCOUNT,
  ACTION_HAVE_ACCOUNT,
  AUTH_EMAIL_LABEL,
  AUTH_PASSWORD_LABEL,
  BRAND,
  ERR_REGISTRATION_FAILED,
  LABEL_COMPANY_NAME,
  LABEL_CONFIRM_PASSWORD,
  LABEL_FIRST_NAME,
  LABEL_LAST_NAME,
  VAL_COMPANY_NAME_REQUIRED,
  VAL_EMAIL_REQUIRED,
  VAL_FIRST_NAME_REQUIRED,
  VAL_LAST_NAME_REQUIRED,
  VAL_PASSWORD_CONFIRM_MISMATCH,
  VAL_PASSWORD_REQUIRED,
} from '@constants/messages';
import kiloLogo from '@assets/kilo-logo.png';

const Register: React.FC = () => {
  const navigate = useNavigate();
  const { versionInfo } = useSystem();
  const { setUser } = useSession();
  const { setOrganization } = useOrganization();
  const { isEnabled } = useFeatureFlags();
  const showCompanyName = isEnabled('enterprise_organizations');

  // Redirect to login if registration is disabled
  useEffect(() => {
    apiService
      .getAuthSettings()
      .then((settings) => {
        if (!settings.registration_enabled) {
          navigate(ROUTES.LOGIN, { replace: true });
        }
      })
      .catch(() => {
        // If settings fetch fails, allow the page to render
      });
  }, [navigate]);

  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [companyName, setCompanyName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [isLoading, setIsLoading] = useState(false);

  const validate = (): boolean => {
    const errors: Record<string, string> = {};

    if (!firstName.trim()) errors.firstName = VAL_FIRST_NAME_REQUIRED;
    if (!lastName.trim()) errors.lastName = VAL_LAST_NAME_REQUIRED;
    if (showCompanyName && !companyName.trim()) errors.companyName = VAL_COMPANY_NAME_REQUIRED;
    if (!email.trim()) errors.email = VAL_EMAIL_REQUIRED;
    if (!password) errors.password = VAL_PASSWORD_REQUIRED;
    if (password && password !== confirmPassword)
      errors.confirmPassword = VAL_PASSWORD_CONFIRM_MISMATCH;

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!validate()) return;

    setIsLoading(true);
    try {
      const result = await apiService.registerAccount({
        email: email.trim(),
        password,
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        companyName: companyName.trim(),
      });

      // Store access token for subsequent API requests
      storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, result.tokens.accessToken);

      // Store refresh token if provided (refresh_token_enabled=true on backend)
      if (result.tokens.refreshToken) {
        storageService.setItem(STORAGE_KEYS.REFRESH_TOKEN, result.tokens.refreshToken);
      } else {
        storageService.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
      }

      // Set session user profile
      setUser(result.user);

      // Set organization context from user profile
      const { user } = result;
      const defaultOrg = user.memberships.find((m) => m.orgId === user.defaultOrgId);
      const firstOrg = user.memberships[0];
      const org = defaultOrg || firstOrg;

      if (org) {
        setOrganization(org.orgId, org.orgName || DEFAULT_ORG_NAME, user.id);
      }

      navigate(ROUTES.HOME);
    } catch {
      setError(ERR_REGISTRATION_FAILED);
    } finally {
      setIsLoading(false);
    }
  };

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
          maxWidth: 480,
          width: '100%',
          textAlign: 'center',
        }}
      >
        {/* Logo and branding */}
        <Box sx={{ mb: 3, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          <img src={kiloLogo} alt={APP_NAME} style={{ maxWidth: '200px', height: 'auto', marginBottom: '8px' }} />
          <Chip label={versionInfo?.edition ?? APP_EDITION} size="small" variant="outlined" sx={{ mb: 2 }} />
          <Box sx={{ display: 'flex', gap: 2, justifyContent: 'center', mb: 1 }}>
            <MuiLink
              href={versionInfo?.documentationUrl}
              target="_blank"
              rel="noopener"
              variant="caption"
            >
              {BRAND.DOCUMENTATION}
            </MuiLink>
            <MuiLink href={versionInfo?.sourceUrl} target="_blank" rel="noopener" variant="caption">
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

        <Typography variant="h5" component="h2" gutterBottom>
          {ACTION_CREATE_ACCOUNT}
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: AUTH_LAYOUT.SPACING_MB }}>
            {error}
          </Alert>
        )}

        <Box component="form" onSubmit={handleSubmit} noValidate>
          <Box display="flex" gap={2}>
            <TextField
              label={LABEL_FIRST_NAME}
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              error={!!fieldErrors.firstName}
              helperText={fieldErrors.firstName}
              fullWidth
              margin="normal"
              autoFocus
            />
            <TextField
              label={LABEL_LAST_NAME}
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              error={!!fieldErrors.lastName}
              helperText={fieldErrors.lastName}
              fullWidth
              margin="normal"
            />
          </Box>

          {showCompanyName && (
            <TextField
              label={LABEL_COMPANY_NAME}
              value={companyName}
              onChange={(e) => setCompanyName(e.target.value)}
              error={!!fieldErrors.companyName}
              helperText={fieldErrors.companyName}
              fullWidth
              margin="normal"
            />
          )}

          <TextField
            label={AUTH_EMAIL_LABEL}
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={!!fieldErrors.email}
            helperText={fieldErrors.email}
            fullWidth
            margin="normal"
          />

          <TextField
            label={AUTH_PASSWORD_LABEL}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={!!fieldErrors.password}
            helperText={fieldErrors.password}
            fullWidth
            margin="normal"
          />

          <TextField
            label={LABEL_CONFIRM_PASSWORD}
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={!!fieldErrors.confirmPassword}
            helperText={fieldErrors.confirmPassword}
            fullWidth
            margin="normal"
          />

          <Button
            type="submit"
            variant="contained"
            fullWidth
            disabled={isLoading}
            sx={{ mt: AUTH_LAYOUT.SPACING_MT, mb: AUTH_LAYOUT.SPACING_MB }}
          >
            {isLoading ? (
              <CircularProgress size={AUTH_LAYOUT.SPINNER_SIZE} />
            ) : (
              ACTION_CREATE_ACCOUNT
            )}
          </Button>

          <Typography variant="body2" textAlign="center">
            {ACTION_HAVE_ACCOUNT}{' '}
            <RouterLink to={ROUTES.LOGIN} style={{ textDecoration: 'none' }}>
              Log in
            </RouterLink>
          </Typography>
        </Box>
      </Paper>
    </Box>
  );
};

export default Register;
