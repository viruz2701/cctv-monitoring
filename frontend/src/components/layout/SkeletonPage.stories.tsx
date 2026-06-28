import type { Meta, StoryObj } from '@storybook/react';
import {
  SkeletonDashboard,
  SkeletonAnalytics,
  SkeletonListPage,
  SkeletonFormPage,
  SkeletonDetailPage,
  SkeletonTechnicianWeek,
  SkeletonComplianceShield,
  SkeletonAdvancedAnalytics,
} from './SkeletonPage';

const meta: Meta<typeof SkeletonDashboard> = {
  title: 'Layout/SkeletonPage',
  decorators: [
    (Story) => (
      <div className="p-4 max-w-6xl mx-auto bg-gray-50 dark:bg-slate-900 min-h-screen">
        <Story />
      </div>
    ),
  ],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'SkeletonPage — скелетоны для разных типов страниц с shimmer-анимацией. ' +
          'Соответствует P1-UX.4: Skeleton на всех страницах.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;

export const DashboardSkeleton: StoryObj = {
  render: () => <SkeletonDashboard />,
};

export const ListSkeleton: StoryObj = {
  render: () => <SkeletonListPage />,
};

export const DetailSkeleton: StoryObj = {
  render: () => <SkeletonDetailPage />,
};

export const FormSkeleton: StoryObj = {
  render: () => <SkeletonFormPage />,
};

export const AnalyticsSkeleton: StoryObj = {
  render: () => <SkeletonAnalytics />,
};

export const TechnicianWeekSkeleton: StoryObj = {
  render: () => <SkeletonTechnicianWeek />,
};

export const ComplianceShieldSkeleton: StoryObj = {
  render: () => <SkeletonComplianceShield />,
};

export const AdvancedAnalyticsSkeleton: StoryObj = {
  render: () => <SkeletonAdvancedAnalytics />,
};
