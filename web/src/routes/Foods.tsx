// Foods, browse and search the food library across all sources, filter by
// provenance, and open any food for the full breakdown.

import { useEffect, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useFoods, useSearchFoods, useCatalogSearch } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState, Spinner } from '@/components/ui'
import { FoodCard } from '@/components/FoodCard'
import { FoodDetailModal } from '@/components/FoodDetailModal'
import { FrequentFoods } from '@/components/FrequentFoods'
import type { FoodDetail } from '@/lib/types'
import { FoodsIcon, SearchIcon } from '@/components/icons'
import { stagger } from '@/lib/motion'

// OpenFoodFacts/TACO/USDA are proper nouns, not translated.
const SOURCES: { labelKey?: string; label?: string; value: string }[] = [
  { labelKey: 'foods.sourceAll', value: '' },
  { labelKey: 'foods.sourceLibrary', value: 'food_library' },
  { label: 'OpenFoodFacts', value: 'openfoodfacts' },
  { label: 'TACO', value: 'taco' },
  { label: 'USDA', value: 'usda' },
]

const CATALOG_PAGE_SIZE = 30

export function Foods() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'library' | 'catalog'>('library')
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  const [source, setSource] = useState('')
  const [catalogLimit, setCatalogLimit] = useState(CATALOG_PAGE_SIZE)
  const [selected, setSelected] = useState<string | null>(null)

  // Debounce the search input so we don't fire a request per keystroke.
  // Also resets catalog pagination, since a new query invalidates the page count.
  useEffect(() => {
    const id = setTimeout(() => {
      setQuery(rawQuery.trim())
      setCatalogLimit(CATALOG_PAGE_SIZE)
    }, 250)
    return () => clearTimeout(id)
  }, [rawQuery])

  const searching = query.length > 0
  const search = useSearchFoods(query)
  const browse = useFoods(source)
  const catalog = useCatalogSearch(query, source, catalogLimit)

  const isLoading =
    tab === 'catalog' ? catalog.isLoading : searching ? search.isLoading : browse.isLoading
  const foods = useMemo(() => {
    if (tab === 'catalog') return catalog.data ?? []
    return (searching ? search.data : browse.data) ?? []
  }, [tab, catalog.data, searching, search.data, browse.data])

  return (
    <div>
      <PageHeader eyebrow={t('foods.eyebrow')} title={t('foods.title')} />

      <div className="mb-5">
        <FrequentFoods />
      </div>

      <div className="mb-4 flex gap-2">
        {(['library', 'catalog'] as const).map((tb) => (
          <button
            key={tb}
            onClick={() => {
              setTab(tb)
              setCatalogLimit(CATALOG_PAGE_SIZE)
            }}
            className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition ${
              tab === tb
                ? 'border-transparent bg-primary text-white'
                : 'border-line bg-surface text-muted hover:text-ink'
            }`}
          >
            {t(tb === 'library' ? 'foods.libraryTab' : 'foods.catalogTab')}
          </button>
        ))}
      </div>

      <div className="relative mb-4">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          <SearchIcon width={18} height={18} />
        </span>
        <input
          value={rawQuery}
          onChange={(e) => setRawQuery(e.target.value)}
          placeholder={t('foods.searchPlaceholder')}
          aria-label={t('foods.searchAriaLabel')}
          className="w-full rounded-full border border-line bg-surface py-2.5 pl-10 pr-4 text-ink outline-none transition focus:border-primary"
        />
      </div>

      {(tab === 'catalog' || !searching) && (
        <div className="mb-6 flex flex-wrap gap-2">
          {SOURCES.map((s) => (
            <button
              key={s.value}
              onClick={() => {
                setSource(s.value)
                setCatalogLimit(CATALOG_PAGE_SIZE)
              }}
              className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                source === s.value
                  ? 'border-transparent bg-primary-soft text-primary'
                  : 'border-line bg-surface text-muted hover:text-ink'
              }`}
            >
              {s.labelKey ? t(s.labelKey) : s.label}
            </button>
          ))}
        </div>
      )}

      {isLoading ? (
        <Spinner label={t('foods.loadingLabel')} />
      ) : !foods.length ? (
        <EmptyState
          icon={<FoodsIcon />}
          title={
            tab === 'catalog'
              ? t('foods.catalogEmptyTitle')
              : searching
                ? t('foods.noMatchesTitle')
                : t('foods.emptyTitle')
          }
          hint={
            tab === 'catalog'
              ? t('foods.catalogEmptyHint')
              : searching
                ? t('foods.noMatchesHint')
                : t('foods.emptyHint')
          }
        />
      ) : (
        <>
          <motion.div
            variants={stagger}
            initial="hidden"
            animate="show"
            className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
          >
            {foods.map((f: FoodDetail) => (
              <FoodCard key={f.food_id} food={f} onClick={() => setSelected(f.food_id)} />
            ))}
          </motion.div>
          {tab === 'catalog' && foods.length >= catalogLimit && (
            <div className="mt-4 flex justify-center">
              <button
                onClick={() => setCatalogLimit((n) => n + CATALOG_PAGE_SIZE)}
                className="rounded-full border border-line bg-surface px-4 py-2 text-sm font-medium text-ink transition hover:border-primary"
              >
                {t('foods.loadMore')}
              </button>
            </div>
          )}
        </>
      )}

      {selected && <FoodDetailModal foodID={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
