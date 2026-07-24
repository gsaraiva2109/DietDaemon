// Foods, browse and search the food library across all sources, filter by
// provenance, and open any food for the full breakdown.

import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useFoods, useSearchFoods, useCatalogSearch } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState, Spinner } from '@/components/ui'
import { FoodCard } from '@/components/FoodCard'
import { FoodDetailModal } from '@/components/FoodDetailModal'
import { CustomFoodModal } from '@/components/CustomFoodModal'
import { FrequentFoods } from '@/components/FrequentFoods'
import { useDemo } from '@/lib/demo'
import type { FoodDetail } from '@/lib/types'
import { FoodsIcon, SearchIcon } from '@/components/icons'
import { stagger } from '@/lib/motion'

// OpenFoodFacts/TACO/USDA are proper nouns, not translated.
const SOURCES: { labelKey?: string; label?: string; value: string }[] = [
  { labelKey: 'foods.sourceAll', value: '' },
  { labelKey: 'foods.sourceCustom', value: 'custom' },
  { label: 'OpenFoodFacts', value: 'openfoodfacts' },
  { label: 'TACO', value: 'taco' },
  { label: 'USDA', value: 'usda' },
]

const CATALOG_PAGE_SIZE = 30

type FoodsTab = 'library' | 'catalog'

// Which query backs `isLoading` depends on the active tab, and on the library
// tab, on whether a search is active. Kept as its own statement (not a nested
// ternary) so each case reads as a plain fact rather than a chain to parse.
function resolveIsLoading(
  tab: FoodsTab,
  searching: boolean,
  catalogLoading: boolean,
  searchLoading: boolean,
  browseLoading: boolean,
): boolean {
  if (tab === 'catalog') return catalogLoading
  return searching ? searchLoading : browseLoading
}

// Empty-state copy depends on the same tab/searching combination as
// isLoading above; resolved together so the title and hint never drift out
// of sync, and so neither is a ternary nested inside another.
function getEmptyStateCopy(tab: FoodsTab, searching: boolean, t: TFunction) {
  if (tab === 'catalog') {
    return { title: t('foods.catalogEmptyTitle'), hint: t('foods.catalogEmptyHint') }
  }
  if (searching) {
    return { title: t('foods.noMatchesTitle'), hint: t('foods.noMatchesHint') }
  }
  return { title: t('foods.emptyTitle'), hint: t('foods.emptyHint') }
}

function FoodsEmptyState({
  tab,
  searching,
  demo,
  onAddCustom,
}: Readonly<{
  tab: FoodsTab
  searching: boolean
  demo: boolean
  onAddCustom: () => void
}>) {
  const { t } = useTranslation()
  const { title, hint } = getEmptyStateCopy(tab, searching, t)
  return (
    <>
      <EmptyState icon={<FoodsIcon />} title={title} hint={hint} />
      {tab === 'library' && (
        <div className="mt-4 flex justify-center">
          <button
            type="button"
            disabled={demo}
            onClick={onAddCustom}
            className="rounded-full border border-line bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:border-primary disabled:opacity-50"
          >
            {t('foods.addCustom')}
          </button>
        </div>
      )}
    </>
  )
}

function FoodsResults({
  foods,
  tab,
  catalogLimit,
  onSelect,
  onLoadMore,
}: Readonly<{
  foods: FoodDetail[]
  tab: FoodsTab
  catalogLimit: number
  onSelect: (foodID: string) => void
  onLoadMore: () => void
}>) {
  const { t } = useTranslation()
  return (
    <>
      <motion.div
        variants={stagger}
        initial="hidden"
        animate="show"
        className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
      >
        {foods.map((f) => (
          <FoodCard key={f.food_id} food={f} onClick={() => onSelect(f.food_id)} />
        ))}
      </motion.div>
      {tab === 'catalog' && foods.length >= catalogLimit && (
        <div className="mt-4 flex justify-center">
          <button
            onClick={onLoadMore}
            className="rounded-full border border-line bg-surface px-4 py-2 text-sm font-medium text-ink transition hover:border-primary"
          >
            {t('foods.loadMore')}
          </button>
        </div>
      )}
    </>
  )
}

export function Foods() {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const [tab, setTab] = useState<FoodsTab>('library')
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  const [source, setSource] = useState('')
  const [catalogLimit, setCatalogLimit] = useState(CATALOG_PAGE_SIZE)
  const [selected, setSelected] = useState<string | null>(null)
  const [customFood, setCustomFood] = useState<FoodDetail | null | 'new'>(null)

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

  const isLoading = resolveIsLoading(tab, searching, catalog.isLoading, search.isLoading, browse.isLoading)
  const foods = useMemo(() => {
    if (tab === 'catalog') return catalog.data ?? []
    return (searching ? search.data : browse.data) ?? []
  }, [tab, catalog.data, searching, search.data, browse.data])

  function selectTab(tb: FoodsTab) {
    setTab(tb)
    if (tb === 'library') setSource('')
    setCatalogLimit(CATALOG_PAGE_SIZE)
  }

  function selectSource(value: string) {
    setSource(value)
    setCatalogLimit(CATALOG_PAGE_SIZE)
  }

  let content: ReactNode
  if (isLoading) {
    content = <Spinner label={t('foods.loadingLabel')} />
  } else if (!foods.length) {
    content = (
      <FoodsEmptyState tab={tab} searching={searching} demo={demo} onAddCustom={() => setCustomFood('new')} />
    )
  } else {
    content = (
      <FoodsResults
        foods={foods}
        tab={tab}
        catalogLimit={catalogLimit}
        onSelect={setSelected}
        onLoadMore={() => setCatalogLimit((n) => n + CATALOG_PAGE_SIZE)}
      />
    )
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <PageHeader eyebrow={t('foods.eyebrow')} title={t('foods.title')} />
        <button
          type="button"
          disabled={demo}
          onClick={() => setCustomFood('new')}
          className="mb-6 rounded-full bg-primary px-4 py-2 text-sm font-semibold text-primary-ink transition hover:brightness-[1.05] disabled:opacity-50"
        >
          {t('foods.addCustom')}
        </button>
      </div>

      <div className="mb-5">
        <FrequentFoods />
      </div>

      <div className="mb-4 flex gap-2">
        {(['library', 'catalog'] as const).map((tb) => (
          <button
            key={tb}
            onClick={() => selectTab(tb)}
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
              onClick={() => selectSource(s.value)}
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

      {content}

      {selected && (
        <FoodDetailModal
          foodID={selected}
          onClose={() => setSelected(null)}
          onEditCustom={(food) => {
            setSelected(null)
            setCustomFood(food)
          }}
        />
      )}
      {customFood && (
        <CustomFoodModal
          food={customFood === 'new' ? undefined : customFood}
          onClose={() => setCustomFood(null)}
          onSaved={(food) => {
            setCustomFood(null)
            setSelected(food.food_id)
          }}
        />
      )}
    </div>
  )
}
