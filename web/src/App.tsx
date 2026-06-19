import { lazy, Suspense } from 'react'
import { AnimatePresence, MotionConfig, motion } from 'framer-motion'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  BrowserRouter,
  Outlet,
  Route,
  Routes,
  useLocation,
} from 'react-router-dom'
import { Toaster } from 'sonner'
import { AuthProvider } from '@/lib/auth'
import { ThemeProvider, useTheme } from '@/lib/theme'
import { DemoProvider } from '@/lib/demo'
import { AppShell } from '@/components/AppShell'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { CommandPalette } from '@/components/CommandPalette'
import { Spinner } from '@/components/ui'
import { easeOut } from '@/lib/motion'
import { Login } from '@/routes/Login'
import { Register } from '@/routes/Register'
import { AuthCallback } from '@/routes/AuthCallback'
import { VerifyEmail } from '@/routes/VerifyEmail'
import { ForgotPassword } from '@/routes/ForgotPassword'
import { ResetPassword } from '@/routes/ResetPassword'

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
const Security = lazy(() => import('@/routes/Security').then(m => ({ default: m.Security })))
const Foods = lazy(() => import('@/routes/Foods').then(m => ({ default: m.Foods })))
const Aliases = lazy(() => import('@/routes/Aliases').then(m => ({ default: m.Aliases })))
const Templates = lazy(() => import('@/routes/Templates').then(m => ({ default: m.Templates })))
const Body = lazy(() => import('@/routes/Body').then(m => ({ default: m.Body })))
const Goals = lazy(() => import('@/routes/Goals').then(m => ({ default: m.Goals })))
const OnboardingWizard = lazy(() =>
  import('@/components/OnboardingWizard').then(m => ({ default: m.OnboardingWizard })),
)

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false },
  },
})

// The authenticated app frame: calm shell, command palette, onboarding, and a
// quick fade-and-rise between routes (keyed by path so the exit runs first).
function AppLayout() {
  const location = useLocation()
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
          <AnimatePresence mode="wait">
            <motion.div
              key={location.pathname}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.26, ease: easeOut }}
            >
              <Outlet />
            </motion.div>
          </AnimatePresence>
        </Suspense>
      </AppShell>
      <CommandPalette />
      <Suspense fallback={null}>
        <OnboardingWizard />
      </Suspense>
    </>
  )
}

function AppRoutes() {
  return (
    <Routes>
      {/* Public auth screens. */}
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route path="/verify-email" element={<VerifyEmail />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />

      {/* Everything else is gated, then wrapped in the app frame. */}
      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/log" element={<LogMeal />} />
          <Route path="/history" element={<History />} />
          <Route path="/history/:mealID" element={<MealDetail />} />
          <Route path="/trends" element={<Trends />} />
          <Route path="/summary" element={<Summary />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/settings/security" element={<Security />} />
          <Route path="/settings/aliases" element={<Aliases />} />
          <Route path="/foods" element={<Foods />} />
          <Route path="/templates" element={<Templates />} />
          <Route path="/body" element={<Body />} />
          <Route path="/body/:tab" element={<Body />} />
          <Route path="/goals" element={<Goals />} />
        </Route>
      </Route>
    </Routes>
  )
}

// Sonner renders in its own portal outside the .dark scope, so it needs the
// theme passed explicitly. Re-renders on toggle, recoloring toasts live.
function ToasterWithTheme() {
  const { theme } = useTheme()
  return <Toaster position="top-center" richColors closeButton theme={theme} />
}

export default function App() {
  return (
    <MotionConfig reducedMotion="user">
      <ThemeProvider>
        <DemoProvider>
          <QueryClientProvider client={queryClient}>
            <BrowserRouter>
              <AuthProvider>
                <ToasterWithTheme />
                <AppRoutes />
              </AuthProvider>
            </BrowserRouter>
          </QueryClientProvider>
        </DemoProvider>
      </ThemeProvider>
    </MotionConfig>
  )
}
