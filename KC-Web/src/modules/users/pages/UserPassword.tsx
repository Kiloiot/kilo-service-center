/**
 * UserPassword Page
 *
 * Dedicated password change page for admin user management.
 * Uses ROUTE_TITLES.USER_DETAIL (intentionally reused per NAV_STRUCTURE.md).
 */

import React, { useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import { useChangePassword, useUser } from "@hooks";
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Typography,
} from "@mui/material";

import { DetailNotFound } from "@components/common/DetailNotFound";
import { NewPasswordFields } from "@components/common/NewPasswordFields";
import { useFeedback } from "@contexts/feedback";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { getErrorMessage } from "@utils/error-message";
import { ROUTES } from "@constants/app";
import {
  MSG_PASSWORD_CHANGED,
  USER_FORM,
  USERS_PAGE,
} from "@constants/messages";
import { userDetailPath } from "@router/paths";
import { componentSpacing } from "@theme/index";

const UserPassword: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();

  // Fetch user to display context
  const {
    data: user,
    isLoading: userLoading,
    isError: userError,
  } = useUser(id || "", {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const changePassword = useChangePassword();

  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const feedback = useFeedback();
  const canSubmit =
    !!newPassword &&
    newPassword === confirmPassword &&
    !changePassword.isPending;

  const handleChangePassword = async () => {
    if (!id || !canSubmit) return;
    try {
      await changePassword.mutateAsync({ id, password: newPassword });
      setNewPassword("");
      setConfirmPassword("");
      feedback.success(MSG_PASSWORD_CHANGED);
    } catch (error) {
      feedback.error(
        getErrorMessage(error, USER_FORM.ERR_CHANGE_PASSWORD_FAILED),
      );
    }
  };

  const handleBackToDetail = () => {
    if (id) {
      navigate(userDetailPath(id));
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
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          minHeight: componentSpacing.stateView.pageMinHeight,
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (userError || !user) {
    return (
      <DetailNotFound
        message={USERS_PAGE.ERR_NOT_FOUND}
        backLabel={USERS_PAGE.BACK_TO_LIST}
        onBack={() => navigate(ROUTES.USERS)}
      />
    );
  }

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
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

      <Card sx={{ maxWidth: componentSpacing.formCard.maxWidth }}>
        <CardContent>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            <NewPasswordFields
              password={newPassword}
              confirmation={confirmPassword}
              onPasswordChange={setNewPassword}
              onConfirmationChange={setConfirmPassword}
              passwordLabel={USER_FORM.LABEL_PASSWORD}
              confirmationLabel={USER_FORM.LABEL_CONFIRM_PASSWORD}
              autoFocus
            />
            <Box display="flex" gap={2} mt={2}>
              <Button
                variant="contained"
                onClick={handleChangePassword}
                disabled={!canSubmit}
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
