// Hover, focus and selection highlight: one surface for every interactive
// list, table and menu, with its text in the surface's contrast color.

import {
  autocompleteClasses,
  chipClasses,
  iconButtonClasses,
  linkClasses,
  listItemButtonClasses,
  listItemIconClasses,
  listItemTextClasses,
  menuItemClasses,
  paginationItemClasses,
  tableCellClasses,
  tableRowClasses,
  typographyClasses,
} from "@mui/material";
import type { CSSObject } from "@mui/material/styles";

export interface HighlightPalette {
  main: string;
  contrastText: string;
  secondaryText: string;
}

interface StateClasses {
  selected: string;
  focusVisible: string;
}

const ARIA_SELECTED = '[aria-selected="true"]';

const anyOf = (...states: string[]) =>
  states.map((state) => `&${state}`).join(", ");

// These set their own color, so the surface color alone does not reach them.
const INHERITING_DESCENDANTS = [
  tableCellClasses.root,
  typographyClasses.root,
  listItemTextClasses.primary,
  listItemIconClasses.root,
  iconButtonClasses.root,
  linkClasses.root,
]
  .map((className) => `& .${className}`)
  .concat("& a")
  .join(", ");

export const highlightSurface = (highlight: HighlightPalette): CSSObject => ({
  backgroundColor: highlight.main,
  color: highlight.contrastText,
  [INHERITING_DESCENDANTS]: { color: "inherit" },
  [`& .${listItemTextClasses.secondary}`]: { color: highlight.secondaryText },
});

// Selected-and-hovered counts as hovered, as in MUI's own list items.
export const listItemStates = ({ selected, focusVisible }: StateClasses) => ({
  hovered: anyOf(
    ":hover",
    `.${focusVisible}`,
    `.${selected}:hover`,
    `.${selected}.${focusVisible}`,
  ),
  selected: anyOf(`.${selected}`),
});

const listItemRoot = (classes: StateClasses, surface: CSSObject) => {
  const states = listItemStates(classes);
  return { root: { [states.selected]: surface, [states.hovered]: surface } };
};

export const chipHighlight = (highlight: HighlightPalette) => ({
  [`&.${chipClasses.outlined}.${chipClasses.colorDefault}.${chipClasses.clickable}:hover`]:
    highlightSurface(highlight),
});

// Autocomplete options are styled from the listbox, whose selectors outrank the option slot.
export const createHighlightOverrides = (highlight: HighlightPalette) => {
  const surface = highlightSurface(highlight);
  return {
    MuiTableRow: {
      styleOverrides: {
        root: {
          [anyOf(
            `.${tableRowClasses.hover}:hover`,
            `.${tableRowClasses.selected}`,
            `.${tableRowClasses.selected}:hover`,
          )]: surface,
        },
      },
    },
    MuiMenuItem: { styleOverrides: listItemRoot(menuItemClasses, surface) },
    MuiListItemButton: {
      styleOverrides: listItemRoot(listItemButtonClasses, surface),
    },
    MuiAutocomplete: {
      styleOverrides: {
        listbox: {
          [`& .${autocompleteClasses.option}`]: {
            [anyOf(
              `.${autocompleteClasses.focused}`,
              ARIA_SELECTED,
              `${ARIA_SELECTED}.${autocompleteClasses.focused}`,
              `${ARIA_SELECTED}.${autocompleteClasses.focusVisible}`,
            )]: surface,
          },
        },
      },
    },
    MuiPaginationItem: {
      styleOverrides: {
        root: {
          [`&:hover:not(.${paginationItemClasses.selected})`]: surface,
        },
      },
    },
  };
};
