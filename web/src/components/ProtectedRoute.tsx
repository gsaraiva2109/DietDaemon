// Route guard for the authenticated app. checking → spinner; anon → bounce to
// /login (preserving where we were headed via ?next); authed → render the
// matched child route. Demo mode reports as authed, so it passes through.

import { useEffect } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { Spinner } from './ui'

export function ProtectedRoute() {
  const { t } = useTranslation()
  const { status } = useAuth()
  const location = useLocation()

  useEffect(() => {
    if (status === 'checking') void import('@/routes/Dashboard')
  }, [status])

  if (status === 'checking') {
    return (
      <div className="grid min-h-[100dvh] place-items-center">
        <Spinner label={t('protectedRoute.connecting')} />
      </div>
    )
  }

  if (status === 'anon') {
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/login?next=${next}`} replace />
  }

  return <Outlet />
}
