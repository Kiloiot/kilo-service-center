// Centralized Icon Management
// All icons should be imported from this file - never import directly in components

// Dashboard & Overview Icons
export {
  Analytics as AnalyticsIcon,
  Assessment as AssessmentIcon,
  Dashboard as DashboardIcon,
  Speed as SpeedIcon,
} from '@mui/icons-material';

// Base Station Icons
export {
  CellTower as BaseStationIcon,
  GpsFixed as GpsFixedIcon,
  LocationOn as LocationIcon,
  Map as MapIcon,
  Router as RouterIcon,
  SignalCellularAlt as SignalIcon,
} from '@mui/icons-material';

// End Point Icons
export {
  BluetoothConnected as ConnectedIcon,
  DeviceHub as DeviceHubIcon,
  Sensors as EndPointIcon,
  Memory as MemoryIcon,
} from '@mui/icons-material';

// Message Icons
export {
  Message as MessageIcon,
  Queue as QueueIcon,
  CallReceived as ReceiveIcon,
  Schedule as ScheduleIcon,
  Send as SendIcon,
} from '@mui/icons-material';

// Blueprint & Model Icons
export {
  Schema as BlueprintIcon,
  Category as CategoryIcon,
  Code as CodeIcon,
  Description as DescriptionIcon,
} from '@mui/icons-material';

// User & Admin Icons
export {
  AccountCircle as AccountIcon,
  AdminPanelSettings as AdminIcon,
  Business as BusinessIcon,
  VpnKey as KeyIcon,
  Logout as LogoutIcon,
  People as PeopleIcon,
  Person as PersonIcon,
  Security as SecurityIcon,
  VpnKey as VpnKeyIcon, // Alias for API Keys navigation
} from '@mui/icons-material';

// Status Icons
export {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
  Pending as PendingIcon,
  Refresh as RefreshIcon,
  Circle as StatusIcon,
  CheckCircle as SuccessIcon,
  Sync as SyncIcon,
  AccessTime as TimeIcon,
  TrendingDown as TrendingDownIcon,
  TrendingUp as TrendingUpIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

// Action Icons
export {
  Add as AddIcon,
  Archive as ArchiveIcon,
  ArrowBack as ArrowBackIcon,
  Close as CloseIcon,
  CloudUpload as CloudUploadIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  Edit as EditIcon,
  FilterList as FilterListIcon,
  Link as LinkIcon,
  LinkOff as LinkOffIcon,
  OpenInNew as OpenInNewIcon,
  Publish as PublishIcon,
  QueryStats as QueryStatsIcon,
  Save as SaveIcon,
  Search as SearchIcon,
  Upload as UploadIcon,
} from '@mui/icons-material';

// UI & Navigation Icons
export {
  ChevronLeft as ChevronLeftIcon,
  ChevronRight as ChevronRightIcon,
  ExpandLess,
  ExpandLess as ExpandLessIcon,
  ExpandMore,
  ExpandMore as ExpandMoreIcon,
  Menu as MenuIcon,
  MoreVert as MoreIcon,
  NavigateBefore as NavigateBeforeIcon,
  NavigateNext as NavigateNextIcon,
} from '@mui/icons-material';

// Theme Icons
export {
  DarkMode as DarkModeIcon,
  LightMode as LightModeIcon,
  Brightness4 as ThemeIcon,
} from '@mui/icons-material';

// Settings Icons
export {
  Build as BuildIcon,
  Settings as SettingsIcon,
  Tune as TuneIcon,
} from '@mui/icons-material';

// Network Icons
export {
  NetworkCheck as NetworkCheckIcon,
  NetworkWifi as NetworkWifiIcon,
  SignalWifiStatusbar4Bar as StrongSignalIcon,
  SignalWifi2Bar as WeakSignalIcon,
} from '@mui/icons-material';

// Time Icons
export {
  CalendarToday as CalendarTodayIcon,
  History as HistoryIcon,
  Update as UpdateIcon,
} from '@mui/icons-material';

// Additional Icons
export {
  BatteryFull as BatteryFullIcon,
  ContentCopy as ContentCopyIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';

// Export icon size constants
export const ICON_SIZES = {
  small: 16,
  medium: 24,
  large: 32,
  xlarge: 48,
} as const;
