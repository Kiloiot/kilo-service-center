-- No-op by design: the up migration rewrote event device EUIs to the canonical
-- form without keeping the form each writer gave (dashed, lowercase or
-- numeric), so that form cannot be restored. The canonical values are what the
-- previous schema reads too, so leaving them is safe.
SELECT 1;
