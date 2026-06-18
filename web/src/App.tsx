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
import { Dashboard } from '@/routes/Dashboard'
import { LogMeal } from '@/routes/LogMeal'
import { History } from '@/routes/History'
import { MealDetail } from '@/routes/MealDetail'
import { Trends } from '@/routes/Trends'
import { Summary } from '@/routes/Summary'
import { Settings } from '@/routes/Settings'

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
        <AnimatedRoutes />
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
