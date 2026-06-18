import { lazy, Suspense } from 'react'
import { AnimatePresence, MotionConfig, motion } from 'framer-motion'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/lib/auth'
import { ThemeProvider } from '@/lib/theme'
import { DemoProvider } from '@/lib/demo'
import { AppShell } from '@/components/AppShell'
import { TokenGate } from '@/components/TokenGate'
import { CommandPalette } from '@/components/CommandPalette'
import { Spinner } from '@/components/ui'
import { easeOut } from '@/lib/motion'

// Lazy-load all routes so recharts (~300KB) only ships when Trends or
// Summary is visited. Route components use named exports — wrap with
// .then() to feed React.lazy the { default } shape it expects.
const Dashboard = lazy(() => import('@/routes/Dashboard').then(m => ({ default: m.Dashboard })))
const LogMeal = lazy(() => import('@/routes/LogMeal').then(m => ({ default: m.LogMeal })))
const History = lazy(() => import('@/routes/History').then(m => ({ default: m.History })))
const MealDetail = lazy(() => import('@/routes/MealDetail').then(m => ({ default: m.MealDetail })))
const Trends = lazy(() => import('@/routes/Trends').then(m => ({ default: m.Trends })))
const Summary = lazy(() => import('@/routes/Summary').then(m => ({ default: m.Summary })))
const Settings = lazy(() => import('@/routes/Settings').then(m => ({ default: m.Settings })))

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false },
  },
})

function Gate() {
  const { status } = useAuth()

  if (status === 'checking') {
    return (
      <div className="grid min-h-[100dvh] place-items-center">
        <Spinner label="Connecting" />
      </div>
    )
  }
  if (status === 'needs-token') return <TokenGate />

  return (
    <>
      <AppShell>
        <Suspense
          fallback={
            <div className="grid min-h-[60dvh] place-items-center">
              <Spinner />
            </div>
          }
        >
          <AnimatedRoutes />
        </Suspense>
      </AppShell>
      <CommandPalette />
    </>
  )
}

// Animated route transitions: a quick fade-and-rise on navigation, keyed by
// path so AnimatePresence can run the exit before the next screen enters.
function AnimatedRoutes() {
  const location = useLocation()
  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={location.pathname}
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -8 }}
        transition={{ duration: 0.26, ease: easeOut }}
      >
        <Routes location={location}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/log" element={<LogMeal />} />
          <Route path="/history" element={<History />} />
          <Route path="/history/:mealID" element={<MealDetail />} />
          <Route path="/trends" element={<Trends />} />
          <Route path="/summary" element={<Summary />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </motion.div>
    </AnimatePresence>
  )
}

export default function App() {
  return (
    <MotionConfig reducedMotion="user">
      <ThemeProvider>
        <DemoProvider>
          <QueryClientProvider client={queryClient}>
            <BrowserRouter>
              <AuthProvider>
                <Gate />
              </AuthProvider>
            </BrowserRouter>
          </QueryClientProvider>
        </DemoProvider>
      </ThemeProvider>
    </MotionConfig>
  )
}
