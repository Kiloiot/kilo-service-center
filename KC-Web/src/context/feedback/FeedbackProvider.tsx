/**
 * The one place an action reports its outcome: an auto-hiding snackbar above
 * every page, so a result still shows after the view that started it is gone.
 */

import { type ReactNode, useMemo, useState } from "react";

import {
  Alert,
  type AlertColor,
  Snackbar,
  type SnackbarCloseReason,
} from "@mui/material";

import { TIMING_NOTICE_AUTO_HIDE } from "@constants/app";

import { type Feedback, FeedbackContext } from "./feedback-context";

interface Notice {
  severity: AlertColor;
  message: string;
  id: number;
}

export function FeedbackProvider({ children }: { children: ReactNode }) {
  const [notice, setNotice] = useState<Notice | null>(null);

  const feedback = useMemo<Feedback>(() => {
    const show = (severity: AlertColor) => (message: string) =>
      setNotice((previous) => ({
        severity,
        message,
        id: (previous?.id ?? 0) + 1,
      }));
    return {
      success: show("success"),
      warning: show("warning"),
      error: show("error"),
    };
  }, []);

  const close = (_event?: unknown, reason?: SnackbarCloseReason) => {
    // A click elsewhere on the page must not dismiss the result before it is read.
    if (reason === "clickaway") return;
    setNotice(null);
  };

  return (
    <FeedbackContext.Provider value={feedback}>
      {children}
      <Snackbar
        key={notice?.id}
        open={notice !== null}
        autoHideDuration={TIMING_NOTICE_AUTO_HIDE}
        onClose={close}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      >
        <Alert severity={notice?.severity} onClose={() => close()}>
          {notice?.message}
        </Alert>
      </Snackbar>
    </FeedbackContext.Provider>
  );
}
