import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AuthLayout } from '@/components/AuthLayout'

export function NotFound() {
  const { t } = useTranslation()
  return (
    <AuthLayout title={t('notFound.title')} subtitle={t('notFound.subtitle')}>
      <div className="text-center">
        <Link
          to="/"
          className="inline-flex items-center justify-center rounded-full bg-primary px-5 py-2.5 text-sm font-semibold text-primary-ink transition hover:brightness-[1.05]"
        >
          {t('notFound.goHome')}
        </Link>
      </div>
    </AuthLayout>
  )
}
