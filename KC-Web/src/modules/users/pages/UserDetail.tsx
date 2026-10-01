import React, { useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import { useDeleteUser, useUser } from "@hooks";
import { Box, CircularProgress, Grid } from "@mui/material";
import { ConfirmDialog } from "@ui";

import { DetailNotFound } from "@components/common/DetailNotFound";
import { useFeedback } from "@contexts/feedback";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { getErrorMessage } from "@utils/error-message";
import { ROUTES } from "@constants/app";
import { MSG_USER_DELETED, USER_FORM, USERS_PAGE } from "@constants/messages";
import { userPasswordPath } from "@router/paths";
import { componentSpacing } from "@theme/index";

import UserDetailHeader from "../components/UserDetailHeader";
import UserInfoCard from "../components/UserInfoCard";
import UserOrganizationsCard from "../components/UserOrganizationsCard";
import UserProfileCard from "../components/UserProfileCard";
import { useUserProfileForm } from "../hooks";

const UserDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();
  const enabled = isHydrated && isAdmin && Boolean(id);
  const {
    data: user,
    isLoading,
    isError,
    error,
  } = useUser(id || "", { enabled });
  const profile = useUserProfileForm(user);
  const deleteUser = useDeleteUser();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const feedback = useFeedback();

  const backToList = () => navigate(ROUTES.USERS);

  const handleDelete = async () => {
    if (!id) return;
    await deleteUser.mutateAsync(id);
    navigate(ROUTES.USERS);
    feedback.success(MSG_USER_DELETED);
  };

  if (!isHydrated) return null;
  if (!isAdmin) return <Navigate to={ROUTES.HOME} replace />;

  if (isLoading) {
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

  if (isError || !user) {
    return (
      <DetailNotFound
        message={getErrorMessage(error, USERS_PAGE.ERR_NOT_FOUND)}
        backLabel={USERS_PAGE.BACK_TO_LIST}
        onBack={backToList}
      />
    );
  }

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      <UserDetailHeader onBack={backToList} />

      <Grid container spacing={3}>
        <Grid size={componentSpacing.gridSpan.twoThirds}>
          <UserProfileCard
            form={profile.form}
            isSaving={profile.isSaving}
            isDeleting={deleteUser.isPending}
            onChange={profile.updateForm}
            onSave={profile.save}
            onDeleteClick={() => setDeleteDialogOpen(true)}
          />
        </Grid>
        <Grid size={componentSpacing.gridSpan.third}>
          <UserInfoCard
            user={user}
            onChangePassword={() => navigate(userPasswordPath(user.id))}
          />
        </Grid>
        <Grid size={componentSpacing.gridSpan.full}>
          <UserOrganizationsCard userId={user.id} enabled={enabled} />
        </Grid>
      </Grid>

      <ConfirmDialog
        open={deleteDialogOpen}
        title={USERS_PAGE.CONFIRM_DELETE_TITLE}
        message={USER_FORM.CONFIRM_DELETE}
        confirmLabel={USER_FORM.ACTION_DELETE}
        cancelLabel={USER_FORM.ACTION_CANCEL}
        pending={deleteUser.isPending}
        errorFallback={USERS_PAGE.ERR_DELETE_FAILED}
        onConfirm={handleDelete}
        onClose={() => setDeleteDialogOpen(false)}
      />
    </Box>
  );
};

export default UserDetail;
