import { Box, Button, Typography, useTheme } from "@mui/material";

import { BASE_STATIONS_PAGE } from "@constants/messages";
import { SecurityIcon } from "@theme/icons";

interface BaseStationListHeaderProps {
  onAdd: () => void;
}

export default function BaseStationListHeader({
  onAdd,
}: BaseStationListHeaderProps) {
  const theme = useTheme();
  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        mb: theme.spacing(3),
      }}
    >
      <Typography variant="h4" component="h1">
        {BASE_STATIONS_PAGE.TITLE}
      </Typography>
      <Button variant="contained" startIcon={<SecurityIcon />} onClick={onAdd}>
        {BASE_STATIONS_PAGE.ADD_BASE_STATION}
      </Button>
    </Box>
  );
}
