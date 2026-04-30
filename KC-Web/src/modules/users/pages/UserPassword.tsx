/**
 * UserPassword Page
 *
 * Dedicated password change page for admin user management.
 * Uses ROUTE_TITLES.USER_DETAIL (intentionally reused per NAV_STRUCTURE.md).
 */

import React, { useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';

import { useChangePassword, useUser } from '@hooks';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  TextField,
  Typography,
} from '@mui/material';

import { useSession } from '@contexts/SessionContext';
import { ROUTES, UI_TIMING } from '@constants/app';
import { MSG_PASSWORD_CHANGED, USER_FORM, USERS_PAGE } from '@constants/messages';

const UserPassword: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isAdmin, isHydrated } = useSession();

  // Fetch user to display context
  const {
    data: user,
    isLoading: userLoading,
    isError: userError,
  } = useUser(id || '', {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const changePassword = useChangePassword();

  // Form state
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  const handleChangePassword = async () => {
    if (!id) return;

    if (newPassword !== confirmPassword) {
      setPasswordError(USER_FORM.ERR_PASSWORD_MISMATCH);
      return;
    }

    if (!newPassword) {
      setPasswordError(USER_FORM.ERR_PASSWORD_REQUIRED);
      return;
    }

    setPasswordError('');

    try {
      await changePassword.mutateAsync({ id, password: newPassword });
      setNewPassword('');
      setConfirmPassword('');
      setSuccessMessage(MSG_PASSWORD_CHANGED);
      // Redirect to user detail after success
      setTimeout(() => {
        navigate(ROUTES.USER_DETAIL.replace(':id', id));
      }, UI_TIMING.NAVIGATION_DELAY_MS);
    } catch {
      // Error handled by mutation hook
    }
  };

  const handleBackToDetail = () => {
    if (id) {
      navigate(ROUTES.USER_DETAIL.replace(':id', id));
    } else {
      navigate(ROUTES.USERS);
    }
  };

  if (!isHydrated) {
    return null;
  }

  if (!isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (userLoading) {
    return (
      <Box
        sx={{
          pt: 4,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (userError || !user) {
    return (
      <Box sx={{ pt: 4, p: 3 }}>
        <Alert severity="error">{USERS_PAGE.ERR_NOT_FOUND}</Alert>
        <Button sx={{ mt: 2 }} onClick={() => navigate(ROUTES.USERS)}>
          {USERS_PAGE.BACK_TO_LIST}
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          {USER_FORM.ACTION_CHANGE_PASSWORD}
        </Typography>
        <Button variant="outlined" onClick={handleBackToDetail}>
          {USERS_PAGE.BACK_TO_DETAIL}
        </Button>
      </Box>

      {/* User context */}
      <Typography variant="subtitle1" color="text.secondary" sx={{ mb: 3 }}>
        {user.email}
      </Typography>

      {/* Success Message */}
      {successMessage && (
        <Alert severity="success" sx={{ mb: 3 }}>
          {successMessage}
        </Alert>
      )}

      {/* Error from API */}
      {changePassword.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {changePassword.error instanceof Error
            ? changePassword.error.message
            : USER_FORM.ERR_CHANGE_PASSWORD_FAILED}
        </Alert>
      )}

      {/* Validation error */}
      {passwordError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {passwordError}
        </Alert>
      )}

      <Card sx={{ maxWidth: 500 }}>
        <CardContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label={USER_FORM.LABEL_PASSWORD}
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              fullWidth
              autoFocus
            />
            <TextField
              label={USER_FORM.LABEL_CONFIRM_PASSWORD}
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              fullWidth
            />
            <Box display="flex" gap={2} mt={2}>
              <Button
                variant="contained"
                onClick={handleChangePassword}
                disabled={changePassword.isPending || !newPassword}
              >
                {USER_FORM.ACTION_CHANGE_PASSWORD}
              </Button>
              <Button variant="outlined" onClick={handleBackToDetail}>
                {USER_FORM.ACTION_CANCEL}
              </Button>
            </Box>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
};

export default UserPassword;
