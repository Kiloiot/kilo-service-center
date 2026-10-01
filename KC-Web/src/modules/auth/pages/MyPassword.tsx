/**
 * MyPassword Page
 *
 * Self-service password change. The change revokes the user's refresh tokens,
 * so a successful change signs the user out and asks for the new password.
 */

import { useState } from "react";

import { useChangeOwnPassword } from "@hooks";
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
} from "@mui/material";

import { NewPasswordFields } from "@components/common/NewPasswordFields";
import { useFeedback } from "@contexts/feedback";
import { useSignOut } from "@hooks/useSignOut";
import { getErrorMessage } from "@utils/error-message";
import { AUTH_LAYOUT } from "@constants/app";
import { MY_PASSWORD } from "@constants/messages";
import { componentSpacing } from "@theme/index";

export default function MyPassword() {
  const theme = useTheme();

  const [currentPassword, setCurrentPassword] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const changePassword = useChangeOwnPassword();
  const signOut = useSignOut();
  const feedback = useFeedback();
  const isSubmitting = changePassword.isPending;
  const canSubmit =
    !!currentPassword && !!password && password === confirmPassword;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setError(null);
    try {
      await changePassword.mutateAsync({
        currentPassword,
        newPassword: password,
      });
    } catch (err) {
      setError(getErrorMessage(err, MY_PASSWORD.ERR_FAILED));
      return;
    }
    await signOut();
    feedback.success(MY_PASSWORD.SIGNED_OUT);
  };

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "100%",
        p: theme.spacing(3),
      }}
    >
      <Card sx={{ maxWidth: AUTH_LAYOUT.CARD_MAX_WIDTH, width: "100%" }}>
        <CardContent sx={{ p: theme.spacing(3) }}>
          <Typography variant="h5" component="h1" gutterBottom>
            {MY_PASSWORD.TITLE}
          </Typography>

          <Box
            component="form"
            onSubmit={handleSubmit}
            sx={{ mt: theme.spacing(2) }}
          >
            {error && (
              <Alert severity="error" sx={{ mb: theme.spacing(2) }}>
                {error}
              </Alert>
            )}

            <TextField
              fullWidth
              type="password"
              label={MY_PASSWORD.LABEL_CURRENT_PASSWORD}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              disabled={isSubmitting}
              autoFocus
              sx={{ mb: theme.spacing(2) }}
            />

            <Box
              sx={{
                display: "flex",
                flexDirection: "column",
                gap: theme.spacing(2),
                mb: theme.spacing(3),
              }}
            >
              <NewPasswordFields
                password={password}
                confirmation={confirmPassword}
                onPasswordChange={setPassword}
                onConfirmationChange={setConfirmPassword}
                passwordLabel={MY_PASSWORD.LABEL_NEW_PASSWORD}
                confirmationLabel={MY_PASSWORD.LABEL_CONFIRM_PASSWORD}
                disabled={isSubmitting}
              />
            </Box>

            <Button
              type="submit"
              variant="contained"
              fullWidth
              disabled={isSubmitting || !canSubmit}
              startIcon={
                isSubmitting ? (
                  <CircularProgress size={componentSpacing.spinner.button} />
                ) : null
              }
            >
              {MY_PASSWORD.ACTION_SAVE}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}
