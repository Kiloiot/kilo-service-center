import { Box, Link, Typography } from "@mui/material";

import { useSystem } from "@contexts/SystemContext";
import { APP_EDITION, formatPoweredByLabel } from "@constants/app";
import { BRAND } from "@constants/messages";

/** The running edition with its source, documentation and license links. */
export function DisclosureLinks() {
  const { versionInfo } = useSystem();
  const edition = versionInfo?.edition ?? APP_EDITION;
  const links = [
    { label: BRAND.SOURCE, href: versionInfo?.sourceUrl },
    { label: BRAND.DOCUMENTATION, href: versionInfo?.documentationUrl },
    { label: BRAND.LICENSE, href: versionInfo?.licenseUrl },
  ];

  return (
    <Box sx={{ px: 2, py: 1.5, borderTop: 1, borderColor: "divider" }}>
      <Typography variant="caption" display="block" color="text.secondary">
        {edition}
      </Typography>
      <Typography
        variant="caption"
        display="block"
        color="text.secondary"
        sx={{ mb: 0.5 }}
      >
        {formatPoweredByLabel(edition)}
      </Typography>
      <Box sx={{ display: "flex", gap: 1.5 }}>
        {links.map(({ label, href }) => (
          <Link
            key={label}
            href={href}
            target="_blank"
            rel="noopener"
            variant="caption"
          >
            {label}
          </Link>
        ))}
      </Box>
    </Box>
  );
}
