import React from "react";

import type { BlueprintUI } from "@api-types/api";
import {
  Box,
  Card,
  CardContent,
  Divider,
  TextField,
  Typography,
} from "@mui/material";

import { formatDateTime } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { getMonoBody1 } from "@utils/typography";
import { BLUEPRINT_LABELS } from "@constants/messages";

interface BlueprintInfoCardProps {
  blueprint: BlueprintUI;
  isEditing: boolean;
  editedVersion: string;
  onEditedVersionChange: (value: string) => void;
}

const BlueprintInfoCard: React.FC<BlueprintInfoCardProps> = ({
  blueprint,
  isEditing,
  editedVersion,
  onEditedVersionChange,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {BLUEPRINT_LABELS.BLUEPRINT_INFORMATION}
      </Typography>
      <Divider sx={{ mb: 2 }} />

      {isEditing ? (
        <TextField
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          fullWidth
          value={editedVersion}
          onChange={(e) => onEditedVersionChange(e.target.value)}
          sx={{ mb: 2 }}
        />
      ) : (
        <Box sx={{ mb: 2 }}>
          <Typography variant="subtitle2" color="text.secondary">
            {BLUEPRINT_LABELS.LABEL_VERSION}
          </Typography>
          <Typography>{blueprint.version}</Typography>
        </Box>
      )}

      <Box sx={{ mb: 2 }}>
        <Typography variant="subtitle2" color="text.secondary">
          {BLUEPRINT_LABELS.LABEL_TYPE_EUI}
        </Typography>
        <Typography sx={getMonoBody1}>
          {formatEui(blueprint.typeEui)}
        </Typography>
      </Box>

      <Box sx={{ mb: 2 }}>
        <Typography variant="subtitle2" color="text.secondary">
          {BLUEPRINT_LABELS.LABEL_CREATED}
        </Typography>
        <Typography>{formatDateTime(blueprint.createdAt)}</Typography>
      </Box>

      <Box>
        <Typography variant="subtitle2" color="text.secondary">
          {BLUEPRINT_LABELS.LABEL_UPDATED}
        </Typography>
        <Typography>{formatDateTime(blueprint.updatedAt)}</Typography>
      </Box>
    </CardContent>
  </Card>
);

export default BlueprintInfoCard;
