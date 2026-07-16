// Desktop-first app shell: a quiet sidebar on >=md, a bottom bar on mobile.
// Not edge-to-edge sticky; the nav is a calm rail, content breathes.

import { NavLink } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  TodayIcon,
  LogIcon,
  HistoryIcon,
  TrendsIcon,
  SummaryIcon,
  SettingsIcon,
  LeafIcon,
  FoodsIcon,
  TemplateIcon,
  BodyIcon,
  GoalIcon,
  ChatIcon,
} from './icons'
import { UtilityBar, DemoBanner } from './UtilityBar'
import { VerifyEmailBanner } from './VerifyEmailBanner'
import { demoAvailable } from '@/lib/demo'

interface NavItem {
  to: string
  labelKey: string
  Icon: typeof TodayIcon
  end?: boolean
  preload: () => Promise<unknown>
}

// Desktop sidebar, grouped into sections. Mobile keeps a curated 5-item bar;
// the rest stay reachable via the ⌘K palette.
const NAV_GROUPS: { headingKey?: string; items: NavItem[] }[] = [
  {
    items: [
      { to: '/', labelKey: 'today', Icon: TodayIcon, end: true, preload: () => import('@/routes/Dashboard') },
      { to: '/chat', labelKey: 'chat', Icon: ChatIcon, preload: () => import('@/routes/Chat') },
      { to: '/log', labelKey: 'log', Icon: LogIcon, preload: () => import('@/routes/LogMeal') },
      { to: '/history', labelKey: 'history', Icon: HistoryIcon, preload: () => import('@/routes/History') },
    ],
  },
  {
    headingKey: 'discover',
    items: [
      { to: '/foods', labelKey: 'foods', Icon: FoodsIcon, preload: () => import('@/routes/Foods') },
      { to: '/templates', labelKey: 'templates', Icon: TemplateIcon, preload: () => import('@/routes/Templates') },
    ],
  },
  {
    headingKey: 'track',
    items: [
      { to: '/body', labelKey: 'body', Icon: BodyIcon, preload: () => import('@/routes/Body') },
      { to: '/goals', labelKey: 'goals', Icon: GoalIcon, preload: () => import('@/routes/Goals') },
      { to: '/trends', labelKey: 'trends', Icon: TrendsIcon, preload: () => import('@/routes/Trends') },
      { to: '/summary', labelKey: 'summary', Icon: SummaryIcon, preload: () => import('@/routes/Summary') },
    ],
  },
  {
    items: [{ to: '/settings', labelKey: 'settings', Icon: SettingsIcon, preload: () => import('@/routes/Settings') }],
  },
]

const MOBILE_NAV: NavItem[] = [
  { to: '/', labelKey: 'today', Icon: TodayIcon, end: true, preload: () => import('@/routes/Dashboard') },
  { to: '/log', labelKey: 'log', Icon: LogIcon, preload: () => import('@/routes/LogMeal') },
  { to: '/foods', labelKey: 'foods', Icon: FoodsIcon, preload: () => import('@/routes/Foods') },
  { to: '/body', labelKey: 'body', Icon: BodyIcon, preload: () => import('@/routes/Body') },
  { to: '/settings', labelKey: 'more', Icon: SettingsIcon, preload: () => import('@/routes/Settings') },
]

function Brand() {
  return (
    <div className="flex items-center gap-2.5 px-2">
      <span className="grid size-9 place-items-center rounded-xl bg-primary-soft text-primary">
        <LeafIcon />
      </span>
      <span className="text-[15px] font-bold tracking-tight text-ink">DietDaemon</span>
    </div>
  )
}

export function AppShell({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <div className="relative min-h-[100dvh]">
      {/* Calm gradient-mesh backdrop, sage glows, fixed behind everything. */}
      <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
        <div className="absolute -left-32 -top-24 size-[34rem] rounded-full bg-primary/20 blur-[120px]" />
        <div className="absolute right-[-10rem] top-1/3 size-[30rem] rounded-full bg-fiber/15 blur-[130px]" />
        <div className="absolute bottom-[-12rem] left-1/4 size-[32rem] rounded-full bg-carbs/10 blur-[140px]" />
      </div>
      {/* Sidebar, desktop */}
      <aside className="fixed inset-y-0 left-0 z-[1100] hidden w-60 flex-col gap-1 overflow-y-auto border-r border-line bg-surface/60 px-3 py-5 backdrop-blur md:flex">
        <Brand />
        <nav className="mt-6 flex flex-col gap-4">
          {NAV_GROUPS.map((group, gi) => (
            <div key={gi} className="flex flex-col gap-1">
              {group.headingKey && (
                <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-muted/70">
                  {t(`nav.${group.headingKey}`)}
                </p>
              )}
              {group.items.map(({ to, labelKey, Icon, end, preload }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  onMouseEnter={() => { void preload() }}
                  className={({ isActive }) =>
                    `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                      isActive
                        ? 'bg-primary-soft text-primary'
                        : 'text-muted hover:bg-surface-2 hover:text-ink'
                    }`
                  }
                >
                  <Icon />
                  {t(`nav.${labelKey}`)}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
      </aside>

      {/* Content */}
      <main className="px-5 pb-28 pt-6 md:ml-60 md:px-10 md:pb-10 md:pt-8">
        <div className="mx-auto w-full max-w-5xl">
          <UtilityBar />
          {demoAvailable() && <DemoBanner />}
          <VerifyEmailBanner />
          {children}
        </div>
      </main>

      {/* Bottom bar, mobile */}
      <nav className="fixed inset-x-0 bottom-0 z-[1100] flex items-stretch justify-around border-t border-line bg-surface/90 px-2 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden">
        {MOBILE_NAV.map(({ to, labelKey, Icon, end, preload }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            onMouseEnter={() => { void preload() }}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center gap-1 py-2.5 text-[11px] font-medium transition ${
                isActive ? 'text-primary' : 'text-muted'
              }`
            }
          >
            <Icon width={20} height={20} />
            {t(`nav.${labelKey}`)}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
