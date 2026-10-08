import type { SystemUserUI } from "@api-types/api";
import { Button, Card, CardContent, Typography } from "@mui/material";

import { formatRelativeDuration } from "@utils/date-format";
import { USER_FORM } from "@constants/messages";

interface UserInfoCardProps {
  user: SystemUserUI;
  onChangePassword: () => void;
}

export default function UserInfoCard({
  user,
  onChangePassword,
}: UserInfoCardProps) {
  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          {USER_FORM.INFO_TITLE}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {USER_FORM.INFO_CREATED}: {formatRelativeDuration(user.createdAt)}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {USER_FORM.INFO_UPDATED}: {formatRelativeDuration(user.updatedAt)}
        </Typography>
        <Button
          variant="outlined"
          size="small"
          onClick={onChangePassword}
          sx={{ mt: 2 }}
          fullWidth
        >
          {USER_FORM.ACTION_CHANGE_PASSWORD}
        </Button>
      </CardContent>
    </Card>
  );
}
