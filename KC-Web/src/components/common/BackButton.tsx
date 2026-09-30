/** The one "back" control: a small outlined button with the back arrow. */

import { Button, type SxProps, type Theme } from "@mui/material";

import { ArrowBackIcon } from "@theme/icons";

interface BackButtonProps {
  label: string;
  onClick: () => void;
  sx?: SxProps<Theme>;
}

export function BackButton({ label, onClick, sx }: BackButtonProps) {
  return (
    <Button
      variant="outlined"
      size="small"
      startIcon={<ArrowBackIcon />}
      onClick={onClick}
      sx={sx}
    >
      {label}
    </Button>
  );
}
