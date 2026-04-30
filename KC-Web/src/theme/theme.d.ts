import '@mui/material/styles';

declare module '@mui/material/styles' {
  // For theme.typography.monoFontFamily access
  interface Typography {
    monoFontFamily: string;
  }

  interface TypographyOptions {
    monoFontFamily?: string;
  }

  // For variant definitions (if needed)
  interface TypographyVariants {
    monoFontFamily: string;
  }

  interface TypographyVariantsOptions {
    monoFontFamily?: string;
  }
}
