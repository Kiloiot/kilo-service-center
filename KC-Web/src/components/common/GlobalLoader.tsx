import React from 'react';

import { Box, CircularProgress, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';

import { LOADER } from '@constants/messages';

/**
 * Global Loading Component
 * Used as Suspense fallback for route-level code splitting
 */

const GlobalLoader: React.FC = () => {
  const theme = useTheme();

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        backgroundColor: theme.palette.background.default,
      }}
    >
      <CircularProgress size={60} thickness={4} />
      <Typography
        variant="h6"
        sx={{
          mt: theme.spacing(3),
          color: theme.palette.text.secondary,
        }}
      >
        {LOADER.LOADING}
      </Typography>
    </Box>
  );
};

export default GlobalLoader;
