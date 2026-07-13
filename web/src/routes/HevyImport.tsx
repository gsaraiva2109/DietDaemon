// Hevy workout import: set/delete the Hevy API key, then trigger a one-time
// import of past workouts. A settings sub-page, same shape as BackupSettings
// (back link + PageHeader).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useHevyKey, useSetHevyKey, useDeleteHevyKey, useImportHevy } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, Spinner } from '@/components/ui'
import { ChevronLeft } from '@/components/icons'

export function HevyImport() {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const query = useHevyKey()
  const setKey = useSetHevyKey()
  const deleteKey = useDeleteHevyKey()
  const importHevy = useImportHevy()

  const [keyValue, setKeyValue] = useState('')

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('hevyImport.title')} />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          {t('hevyImport.readOnly')}
        </p>
      )}

      {query.isLoading ? (
        <Spinner label={t('hevyImport.loading')} />
      ) : (
        <>
          <Card className="mb-5 p-5">
            {query.data?.has_key && (
              <p className="mb-4 text-sm text-muted">
                {t('hevyImport.keySet')}
              </p>
            )}

            <div className="grid gap-4 sm:grid-cols-1">
              <Field
                label={t('hevyImport.apiKeyLabel')}
                type="password"
                value={keyValue}
                disabled={demo}
                onChange={(e) => setKeyValue(e.target.value)}
                placeholder={t('hevyImport.apiKeyPlaceholder')}
              />
            </div>

            <p className="mt-2 text-xs text-muted">
              {t('hevyImport.getApiKeyPrefix')}{' '}
              <a
                href="https://www.hevy.com/settings?developer"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-ink"
              >
                hevy.com/settings?developer
              </a>{' '}
              {t('hevyImport.getApiKeySuffix')}
            </p>

            <div className="mt-5 flex items-center gap-3">
              <Button onClick={() => setKey.mutate({ key: keyValue })} disabled={demo || setKey.isPending || !keyValue}>
                {setKey.isPending ? t('hevyImport.saving') : t('hevyImport.save')}
              </Button>
              {setKey.isSuccess && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  {t('hevyImport.saved')}
                </motion.span>
              )}
              {setKey.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setKey.error instanceof Error ? setKey.error.message : t('hevyImport.saveFailed')}
                </span>
              )}
            </div>

            {query.data?.has_key && (
              <div className="mt-6 border-t border-line pt-4">
                <p className="mb-3 text-sm text-muted">
                  {t('hevyImport.removeKeyDescription')}
                </p>
                <Button
                  variant="ghost"
                  onClick={() => deleteKey.mutate()}
                  disabled={demo || deleteKey.isPending}
                >
                  {deleteKey.isPending ? t('hevyImport.deleting') : t('hevyImport.deleteKey')}
                </Button>
                {deleteKey.isSuccess && (
                  <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="ml-3 text-sm font-medium text-primary">
                    {t('hevyImport.keyDeleted')}
                  </motion.span>
                )}
                {deleteKey.isError && (
                  <span className="ml-3 text-sm font-medium text-accent" role="alert">
                    {deleteKey.error instanceof Error ? deleteKey.error.message : t('hevyImport.deleteFailed')}
                  </span>
                )}
              </div>
            )}
          </Card>

          <Card className="p-5">
            <div className="flex items-center justify-between gap-4">
              <div>
                <h2 className="font-semibold text-ink">{t('hevyImport.importWorkouts')}</h2>
                <p className="mt-0.5 text-sm text-muted">
                  {t('hevyImport.importDescription')}
                </p>
              </div>
              <Button
                onClick={() => importHevy.mutate()}
                disabled={demo || importHevy.isPending || !query.data?.has_key}
              >
                {importHevy.isPending ? t('hevyImport.importing') : t('hevyImport.importNow')}
              </Button>
            </div>
            {importHevy.isSuccess && (
              <p className="mt-3 text-sm font-medium text-primary">
                {t('hevyImport.importSuccess', {
                  imported: importHevy.data.imported,
                  skipped: importHevy.data.skipped_duplicates,
                  total: importHevy.data.total,
                })}
              </p>
            )}
            {importHevy.isError && (
              <p className="mt-3 text-sm font-medium text-accent" role="alert">
                {importHevy.error instanceof Error ? importHevy.error.message : t('hevyImport.importFailed')}
              </p>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
