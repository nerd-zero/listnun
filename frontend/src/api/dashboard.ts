// Typed edge over the generated dashboard client; responses arrive
// camelCased by the shared mutator.
import { getDashboard } from './generated/endpoints/dashboard/dashboard';

export type DashboardPeriod = 7 | 30 | 90;

export interface DashboardDay {
  date: string;
  delivered: number;
  bounced: number;
  failed: number;
  newSubscribers: number;
}

export interface DashboardOverview {
  days: DashboardDay[];
  subscribers: number;
  unsubscribed: number;
  campaignsStarted: number;
  running: number;
  queued: number;
  staleDrafts: number;
  bounceHotspot: { listId: number; listName: string; hard: number } | null;
  sent: number;
  bounced: number;
  new: number;
  bounceRate: number;
  asOf: string;
}

// eslint-disable-next-line import/prefer-default-export
export const getDashboardOverview = (days: DashboardPeriod) => (
  getDashboard().getDashboardOverview({ days }) as unknown as Promise<DashboardOverview>
);
