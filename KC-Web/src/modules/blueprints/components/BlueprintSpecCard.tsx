import React from "react";

import { Card, CardContent, Divider, Paper, Typography } from "@mui/material";

import { BLUEPRINT_SPEC_ROWS, JSON_PREVIEW } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import SpecJsonField from "./SpecJsonField";

interface BlueprintSpecCardProps {
  specJson: object;
  isEditing: boolean;
  editedSpec: string;
  onEditedSpecChange: (value: string) => void;
}

const BlueprintSpecCard: React.FC<BlueprintSpecCardProps> = ({
  specJson,
  isEditing,
  editedSpec,
  onEditedSpecChange,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {BLUEPRINT_LABELS.LABEL_SPEC_JSON}
      </Typography>
      <Divider sx={{ mb: 2 }} />

      {isEditing ? (
        <SpecJsonField
          value={editedSpec}
          onChange={onEditedSpecChange}
          rows={BLUEPRINT_SPEC_ROWS.DETAIL}
        />
      ) : (
        <Paper
          sx={(theme) => ({
            p: 2,
            bgcolor: "grey.900",
            color: "grey.100",
            fontFamily: theme.typography.monoFontFamily,
            fontSize: theme.typography.body2.fontSize,
            overflow: "auto",
            maxHeight: componentSpacing.specPreview.maxHeight,
          })}
        >
          <pre style={{ margin: 0 }}>
            {JSON.stringify(specJson, null, JSON_PREVIEW.INDENT)}
          </pre>
        </Paper>
      )}
    </CardContent>
  </Card>
);

export default BlueprintSpecCard;
