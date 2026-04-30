/**
 * MyPassword Page
 *
 * Self-service password change page for authenticated users.
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  TextField,
  Typography,
  useTheme,
} from '@mui/material';

import { apiService } from '@services/api';
import { ROUTES, UI_TIMING } from '@constants/app';
import { MY_PASSWORD } from '@constants/messages';

export default function MyPassword() {
  const theme = useTheme();
  const navigate = useNavigate();

  const [currentPassword, setCurrentPassword] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    // Validate passwords match (policy enforcement is server-side)
    if (password !== confirmPassword) {
      setError(MY_PASSWORD.ERR_MISMATCH);
      return;
    }

    setIsSubmitting(true);

    try {
      await apiService.changeOwnPassword(currentPassword, password);
      setSuccess(true);
      setCurrentPassword('');
      setPassword('');
      setConfirmPassword('');
      // Redirect after success message shows
      setTimeout(() => {
        navigate(ROUTES.HOME);
      }, UI_TIMING.NAVIGATION_DELAY_MS);
    } catch {
      setError(MY_PASSWORD.ERR_FAILED);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100%',
        p: theme.spacing(3),
      }}
    >
      <Card sx={{ maxWidth: 400, width: '100%' }}>
        <CardContent sx={{ p: theme.spacing(3) }}>
          <Typography variant="h5" component="h1" gutterBottom>
            {MY_PASSWORD.TITLE}
          </Typography>

          <Box component="form" onSubmit={handleSubmit} sx={{ mt: theme.spacing(2) }}>
            {error && (
              <Alert severity="error" sx={{ mb: theme.spacing(2) }}>
                {error}
              </Alert>
            )}

            {success && (
              <Alert severity="success" sx={{ mb: theme.spacing(2) }}>
                {MY_PASSWORD.SUCCESS}
              </Alert>
            )}

            <TextField
              fullWidth
              type="password"
              label={MY_PASSWORD.LABEL_CURRENT_PASSWORD}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              disabled={isSubmitting || success}
              autoFocus
              sx={{ mb: theme.spacing(2) }}
            />

            <TextField
              fullWidth
              type="password"
              label={MY_PASSWORD.LABEL_NEW_PASSWORD}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={isSubmitting || success}
              sx={{ mb: theme.spacing(2) }}
            />

            <TextField
              fullWidth
              type="password"
              label={MY_PASSWORD.LABEL_CONFIRM_PASSWORD}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={isSubmitting || success}
              sx={{ mb: theme.spacing(3) }}
            />

            <Button
              type="submit"
              variant="contained"
              fullWidth
              disabled={
                isSubmitting || success || !currentPassword || !password || !confirmPassword
              }
              startIcon={isSubmitting ? <CircularProgress size={20} /> : null}
            >
              {MY_PASSWORD.ACTION_SAVE}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}
