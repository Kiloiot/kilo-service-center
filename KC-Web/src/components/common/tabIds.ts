import { TAB_A11Y_ID } from "@constants/app";

export const tabId = (prefix: string, value: string) =>
  `${prefix}${TAB_A11Y_ID.TAB}${value}`;

export const tabPanelId = (prefix: string, value: string) =>
  `${prefix}${TAB_A11Y_ID.PANEL}${value}`;
