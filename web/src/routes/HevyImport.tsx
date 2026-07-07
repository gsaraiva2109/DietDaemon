// Hevy workout import: set/delete the Hevy API key, then trigger a one-time
// import of past workouts. A settings sub-page, same shape as BackupSettings
// (back link + PageHeader).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useHevyKey, useSetHevyKey, useDeleteHevyKey, useImportHevy } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, Spinner } from '@/components/ui'
import { ChevronLeft } from '@/components/icons'

export function HevyImport() {
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
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Hevy Workout Import" />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Hevy integration is read only here.
        </p>
      )}

      {query.isLoading ? (
        <Spinner label="Loading Hevy settings" />
      ) : (
        <>
          <Card className="mb-5 p-5">
            {query.data?.has_key && (
              <p className="mb-4 text-sm text-muted">
                Hevy API key is set.
              </p>
            )}

            <div className="grid gap-4 sm:grid-cols-1">
              <Field
                label="API Key"
                type="password"
                value={keyValue}
                disabled={demo}
                onChange={(e) => setKeyValue(e.target.value)}
                placeholder="Hevy API key"
              />
            </div>

            <p className="mt-2 text-xs text-muted">
              Get your API key at{' '}
              <a
                href="https://www.hevy.com/settings?developer"
                target="_blank"
                rel="noopener noreferrer"
                className="underline hover:text-ink"
              >
                hevy.com/settings?developer
              </a>{' '}
              (requires Hevy Pro).
            </p>

            <div className="mt-5 flex items-center gap-3">
              <Button onClick={() => setKey.mutate({ key: keyValue })} disabled={demo || setKey.isPending || !keyValue}>
                {setKey.isPending ? 'Saving…' : 'Save'}
              </Button>
              {setKey.isSuccess && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  Saved.
                </motion.span>
              )}
              {setKey.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setKey.error instanceof Error ? setKey.error.message : 'Failed to save'}
                </span>
              )}
            </div>

            {query.data?.has_key && (
              <div className="mt-6 border-t border-line pt-4">
                <p className="mb-3 text-sm text-muted">
                  Remove the stored Hevy API key.
                </p>
                <Button
                  variant="ghost"
                  onClick={() => deleteKey.mutate()}
                  disabled={demo || deleteKey.isPending}
                >
                  {deleteKey.isPending ? 'Deleting…' : 'Delete key'}
                </Button>
                {deleteKey.isSuccess && (
                  <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="ml-3 text-sm font-medium text-primary">
                    Key deleted.
                  </motion.span>
                )}
                {deleteKey.isError && (
                  <span className="ml-3 text-sm font-medium text-accent" role="alert">
                    {deleteKey.error instanceof Error ? deleteKey.error.message : 'Failed to delete'}
                  </span>
                )}
              </div>
            )}
          </Card>

          <Card className="p-5">
            <div className="flex items-center justify-between gap-4">
              <div>
                <h2 className="font-semibold text-ink">Import workouts</h2>
                <p className="mt-0.5 text-sm text-muted">
                  Pull in your past workouts from Hevy. Duplicates are skipped
                  automatically.
                </p>
              </div>
              <Button
                onClick={() => importHevy.mutate()}
                disabled={demo || importHevy.isPending || !query.data?.has_key}
              >
                {importHevy.isPending ? 'Importing…' : 'Import now'}
              </Button>
            </div>
            {importHevy.isSuccess && (
              <p className="mt-3 text-sm font-medium text-primary">
                Imported {importHevy.data.imported} workouts (
                {importHevy.data.skipped_duplicates} duplicates skipped,{' '}
                {importHevy.data.total} total).
              </p>
            )}
            {importHevy.isError && (
              <p className="mt-3 text-sm font-medium text-accent" role="alert">
                {importHevy.error instanceof Error ? importHevy.error.message : 'Import failed'}
              </p>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
