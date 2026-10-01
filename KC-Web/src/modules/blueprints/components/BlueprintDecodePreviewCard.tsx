import React from "react";

import type { DecodePreviewResponse } from "@api-types/api";
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  Paper,
  TextField,
  Typography,
} from "@mui/material";

import { formatDecodedPayload } from "@utils/formatters";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { CheckCircleIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

interface BlueprintDecodePreviewCardProps {
  testPayload: string;
  testFormatId: number;
  decodeResult: DecodePreviewResponse | null;
  isPending: boolean;
  onTestPayloadChange: (value: string) => void;
  onTestFormatIdChange: (value: number) => void;
  onRun: () => void;
}

const BlueprintDecodePreviewCard: React.FC<BlueprintDecodePreviewCardProps> = ({
  testPayload,
  testFormatId,
  decodeResult,
  isPending,
  onTestPayloadChange,
  onTestFormatIdChange,
  onRun,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {BLUEPRINT_LABELS.TEST_DECODE}
      </Typography>
      <Divider sx={{ mb: 2 }} />

      <TextField
        label={BLUEPRINT_LABELS.LABEL_TEST_DATA}
        fullWidth
        value={testPayload}
        onChange={(e) => onTestPayloadChange(e.target.value)}
        placeholder={BLUEPRINT_LABELS.PLACEHOLDER_HEX}
        helperText={BLUEPRINT_LABELS.HELPER_TEST_DATA}
        sx={{ mb: 2 }}
      />

      <TextField
        label={BLUEPRINT_LABELS.LABEL_FORMAT_ID}
        type="number"
        value={testFormatId}
        onChange={(e) => onTestFormatIdChange(parseInt(e.target.value) || 0)}
        sx={{ mb: 2, width: componentSpacing.compactInput.width }}
      />

      <Button
        variant="contained"
        size="small"
        onClick={onRun}
        disabled={!testPayload || isPending}
        fullWidth
      >
        {isPending ? (
          <CircularProgress size={componentSpacing.spinner.button} />
        ) : (
          BLUEPRINT_LABELS.TEST_DECODE
        )}
      </Button>

      {decodeResult && (
        <Paper
          sx={(theme) => ({
            mt: 2,
            p: 2,
            bgcolor: decodeResult.success ? "success.dark" : "error.dark",
            color: "common.white",
            fontFamily: theme.typography.monoFontFamily,
            fontSize: theme.typography.body2.fontSize,
          })}
        >
          {decodeResult.success ? (
            <>
              <Box
                sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}
              >
                <CheckCircleIcon fontSize="small" />
                <Typography variant="body2">
                  {BLUEPRINT_LABELS.DECODE_SUCCESS}
                </Typography>
              </Box>
              <Typography component="pre" sx={{ whiteSpace: "pre-wrap" }}>
                {decodeResult.decodedData
                  ? formatDecodedPayload(decodeResult.decodedData)
                  : BLUEPRINT_LABELS.NO_DECODE_RESULT}
              </Typography>
            </>
          ) : (
            <>
              <Typography variant="body2" gutterBottom>
                {BLUEPRINT_LABELS.DECODE_FAILED}
              </Typography>
              {decodeResult.errorCode && (
                <Typography variant="body2">
                  {BLUEPRINT_LABELS.ERROR_CODE_PREFIX} {decodeResult.errorCode}
                </Typography>
              )}
              {decodeResult.errorDetail && (
                <Typography variant="body2">
                  {decodeResult.errorDetail}
                </Typography>
              )}
            </>
          )}
        </Paper>
      )}
    </CardContent>
  </Card>
);

export default BlueprintDecodePreviewCard;
