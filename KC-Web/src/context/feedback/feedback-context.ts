import { createContext } from "react";

/** Reports the outcome of an action the user started. */
export interface Feedback {
  success: (message: string) => void;
  warning: (message: string) => void;
  error: (message: string) => void;
}

export const FeedbackContext = createContext<Feedback | null>(null);
