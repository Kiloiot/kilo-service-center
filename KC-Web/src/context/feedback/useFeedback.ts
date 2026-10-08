import { useContext } from "react";

import { FEEDBACK_ERRORS } from "@constants/messages";

import { type Feedback, FeedbackContext } from "./feedback-context";

export function useFeedback(): Feedback {
  const feedback = useContext(FeedbackContext);
  if (!feedback) throw new Error(FEEDBACK_ERRORS.CONTEXT_REQUIRED);
  return feedback;
}
