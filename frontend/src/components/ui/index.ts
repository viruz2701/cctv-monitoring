// UI Components
export { Card, CardHeader, CardContent as CardBody, CardFooter } from './Card';
export { Badge, StatusBadge, HealthBadge, PriorityBadge, TicketStatusBadge, RoleBadge } from './Badge';
export { Button, IconButton } from './Button';
export { StatsCard, MiniStatsCard } from './StatsCard';
export { Table, Pagination } from './Table';
export { Modal, ConfirmModal } from './Modal';
export { Input, SearchInput, Select, Textarea } from './Input';
export { ToastProvider, useToast } from './Toast';
export { VirtualTable } from './VirtualTable';
export { PartCard } from './PartCard';
export { Gauge } from './Gauge';
export { SLAProgress } from './SLAProgress';
export { LiveSLATimer } from './LiveSLATimer';
export { Timeline } from './Timeline';
export { Tabs } from './Tabs';
export { QRCode } from './QRCode';
export { FileUpload } from './FileUpload';
export { ImportWizard, exportData } from './ImportWizard';
export { WorkOrderPrintView } from './WorkOrderPrintView';
export type { WorkOrderPrintViewProps, PartWithCost } from './WorkOrderPrintView';
export * from './AdvancedSearch';
export { MapModal } from './MapModal';
export * from './Modal';
export * from './Button';
export * from './Input';
export * from './Toast';
export { CommandPalette } from './CommandPalette';
export { default as OnboardingTour } from './OnboardingTour';
export { EmptyState } from './EmptyState';
export { Alert } from './Alert';
export { Notification, NotificationList } from './Notification';
export { VisuallyHidden } from './VisuallyHidden';
export {
    Skeleton,
    SkeletonLine,
    SkeletonAvatar,
    SkeletonStatsCard,
    SkeletonCard,
    SkeletonTable,
    SkeletonChart,
    SkeletonFilterBar,
    SkeletonPage,
    SkeletonProfileField,
    SkeletonNotification,
} from './Skeleton';
export { SavedViews } from './SavedViews';
export type { FilterState } from './SavedViews';
export { VideoTutorialCard } from './VideoTutorialCard';
export type { TutorialVideo } from './VideoTutorialCard';
export { ThemeCustomizer } from './ThemeCustomizer';
export { ProgressBar } from './ProgressBar';
export { BulkProgressModal } from './BulkProgressModal';
export type { BulkProgressItem, BulkProgressState, BulkItemStatus } from './BulkProgressModal';
export { Breadcrumbs } from './Breadcrumbs';
export { Tooltip } from './Tooltip';
export { InfoTooltip } from './InfoTooltip';
export { Dropdown } from './Dropdown';
export type { DropdownItem } from './Dropdown';
export { LazyImage } from './LazyImage';

// ── React.memo — оптимизация тяжёлых компонентов ─────────────────────
import React from 'react';
import { DataGrid as DataGridBase } from './DataGrid';
export const DataGrid = React.memo(DataGridBase) as typeof DataGridBase;
