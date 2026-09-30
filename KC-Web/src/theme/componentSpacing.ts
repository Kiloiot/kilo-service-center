/**
 * Component-specific spacing tokens
 * Use via componentSpacing.{component}.{property}
 */
export const componentSpacing = {
  pagination: {
    containerMt: 2, // Container top margin (theme.spacing units)
    containerGap: 2, // Main flex gap (theme.spacing units)
    controlGap: 1, // Inner control gap (theme.spacing units)
    selectMinWidth: 80, // Page size select min width (px)
  },
  dataCard: {
    padding: 3, // Card padding (theme.spacing units)
    headerGap: 2, // Gap between header items (theme.spacing units)
    headerMb: 2, // Space below the header (theme.spacing units)
    chipGap: 1, // Gap between chips of a chip row (theme.spacing units)
  },
  infoGrid: {
    columnGap: 3, // Gap between columns (theme.spacing units)
    rowGap: 1.5, // Gap between rows (theme.spacing units)
    minColumnWidth: 180, // Narrowest column before wrapping (px)
  },
  miniBarChart: {
    height: 72, // Chart height (px)
    barGap: 0.25, // Gap between bars (theme.spacing units)
    minBarHeight: 2, // Visible height of a zero bar (px)
    barRadius: 0.5, // Bar corner radius (theme.shape units)
  },
  cardSection: {
    sectionGap: 3, // Gap between cards of a section (theme.spacing units)
  },
  statCard: {
    iconPadding: 1.5, // Padding of the icon tile (theme.spacing units)
    iconRadius: 2, // Icon tile corner radius (theme.shape units)
    trendGap: 1, // Space above the trend chip (theme.spacing units)
    gridGap: 3, // Gap between the cards of a summary row (theme.spacing units)
    rowMb: 3, // Space below a summary row (theme.spacing units)
    gridSize: { xs: 12, sm: 6, md: 3 }, // Columns of one card per breakpoint (of 12)
  },
  stationMap: {
    popupGap: 0.5, // Gap between the lines of a pin popup (theme.spacing units)
  },
  gridSpan: {
    full: 12, // Every breakpoint (of 12 columns)
    half: { xs: 12, md: 6 },
    halfFromSm: { xs: 12, sm: 6 },
    third: { xs: 12, md: 4 },
    twoThirds: { xs: 12, md: 8 },
    quarter: { xs: 12, md: 3 },
    threeQuarters: { xs: 12, md: 9 },
    fiveTwelfths: { xs: 12, md: 5 },
  },
  stateView: {
    minHeight: 300, // Empty and error states (px)
    listMinHeight: 200, // Loading state of a list page (px)
    pageMinHeight: 400, // Loading or error state of a whole page (px)
    messageMaxWidth: 400, // Message line under the title (px)
    detailsMaxWidth: 600, // Stack trace panel of an error state (px)
    boundaryDetailsMaxWidth: 800, // Stack trace panel of the app error boundary (px)
    iconSize: 64, // Icon above an empty or no-access state (px)
    emptyIconOpacity: 0.6,
    errorIconOpacity: 0.8,
  },
  resultIcon: {
    size: 48, // Success or placeholder icon heading a card or dialog body (px)
  },
  headerIcon: {
    size: 40, // Icon beside a detail page title (px)
  },
  cardTitle: {
    fontWeight: 600, // Card and panel titles
  },
  statusPanel: {
    minHeight: 360, // Dashboard service center status card (px)
  },
  spinner: {
    inline: 16, // Beside a line of text (px)
    buttonSmall: 18, // Inside a small button (px)
    button: 20, // Inside a medium button (px)
    section: 24, // Loading a card, table or form (px)
    page: 60, // Loading the whole app (px)
    pageThickness: 4, // Stroke of the page spinner (px)
  },
  textArea: {
    compactRows: 2, // Notes and short descriptions
    rows: 3, // Descriptions
    tallRows: 4, // Submission descriptions
  },
  formCard: {
    maxWidth: 500, // Form card on an admin page (px)
    registerMaxWidth: 480, // Registration card (px)
    pageMaxWidth: 800, // Full-page form (px)
  },
  userMenu: {
    labelMaxWidth: 160, // Signed-in user's email before it truncates (px)
    menuMinWidth: 200, // Open menu (px)
  },
  truncatedCell: {
    maxWidth: 200, // Table cell text before it truncates (px)
  },
  tableValue: {
    unbroken: "nowrap", // white-space of an EUI, opId, queId or timestamp: never wraps inside the value
    breakAnywhere: "break-all", // word-break of hex user data: wraps between any two digits
  },
  compactInput: {
    width: 120, // Short key or number input (px)
  },
  selectField: {
    minWidth: 120, // Short option select (px)
    wideMinWidth: 250, // Select of named records (px)
  },
  dateInput: {
    minWidth: 150, // Date text field (px)
  },
  specPreview: {
    maxHeight: 500, // Scrollable blueprint spec (px)
  },
  livePulse: {
    duration: "2s", // One fade cycle of a live indicator
    dimOpacity: 0.6, // Opacity at the dimmest point of the cycle
  },
  downlinkForm: {
    formatWidth: 150, // Format field width (px)
    priorityWidth: 200, // Priority field width (px)
  },
  focusRing: {
    width: 2, // Keyboard focus outline width (px)
    offset: 2, // Outline gap outside the control (px)
    insetOffset: -2, // Outline drawn inside, for controls in a clipping container (px)
  },
  button: {
    radius: 8, // Corner radius (px)
    medium: {
      minHeight: 36, // Height of every medium and large button (px)
      paddingX: 16, // Horizontal padding (px)
      fontSize: 14, // Label size (px)
      iconSize: 18, // Start and end icon size (px)
    },
    small: {
      minHeight: 30, // Height of every small button (px)
      paddingX: 12, // Horizontal padding (px)
      fontSize: 13, // Label size (px)
      iconSize: 16, // Start and end icon size (px)
    },
  },
  iconButton: {
    medium: {
      padding: 8, // Padding around the icon: 36px square with the icon (px)
      iconSize: 20, // Icon size (px)
    },
    small: {
      padding: 6, // Padding around the icon: 30px square with the icon (px)
      iconSize: 18, // Icon size (px)
    },
  },
  tabs: {
    minHeight: 36, // Height of the tab bar and every tab, with or without an icon (px)
    paddingX: 16, // Horizontal tab padding (px)
    fontSize: 14, // Label size (px)
    iconSize: 18, // Tab icon size (px)
    iconGap: 1.5, // Gap between a tab's icon and label (theme.spacing units)
    indicatorHeight: 2, // Selected tab underline (px)
    panelPt: 3, // Space between the tab bar and its panel (theme.spacing units)
  },
  userMenuTrigger: {
    paddingX: 1.5, // Horizontal padding (theme.spacing units)
    paddingY: 1, // Vertical padding (theme.spacing units)
    fontSize: 16, // Label size the navigation was drawn with (px)
    iconSize: 20, // Account icon size (px)
    hoverLift: -1, // Vertical shift on hover (px)
    hoverShadowOffsetY: 4, // Hover shadow offset (px)
    hoverShadowBlur: 12, // Hover shadow blur (px)
    hoverShadowOpacity: { light: 0.1, dark: 0.3 }, // Hover shadow strength per mode
    transition: "all 0.2s ease-in-out", // Hover and focus transition
  },
} as const;
