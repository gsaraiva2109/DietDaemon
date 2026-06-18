import { MotionConfig } from 'framer-motion'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/lib/auth'
import { ThemeProvider } from '@/lib/theme'
import { DemoProvider } from '@/lib/demo'
import { AppShell } from '@/components/AppShell'
import { TokenGate } from '@/components/TokenGate'
import { Spinner } from '@/components/ui'
import { Dashboard } from '@/routes/Dashboard'
import { LogMeal } from '@/routes/LogMeal'
import { History } from '@/routes/History'
import { MealDetail } from '@/routes/MealDetail'
import { Trends } from '@/routes/Trends'
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
    <AppShell>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/log" element={<LogMeal />} />
        <Route path="/history" element={<History />} />
        <Route path="/history/:mealID" element={<MealDetail />} />
        <Route path="/trends" element={<Trends />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </AppShell>
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
