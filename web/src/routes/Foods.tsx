// Foods — browse and search the food library across all sources, filter by
// provenance, and open any food for the full breakdown.

import { useEffect, useMemo, useState } from 'react'
import { motion } from 'framer-motion'
import { useFoods, useSearchFoods } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { EmptyState, Spinner } from '@/components/ui'
import { FoodCard } from '@/components/FoodCard'
import { FoodDetailModal } from '@/components/FoodDetailModal'
import { FrequentFoods } from '@/components/FrequentFoods'
import type { FoodDetail } from '@/lib/types'
import { FoodsIcon, SearchIcon } from '@/components/icons'
import { stagger } from '@/lib/motion'

const SOURCES: { label: string; value: string }[] = [
  { label: 'All', value: '' },
  { label: 'Library', value: 'food_library' },
  { label: 'OpenFoodFacts', value: 'openfoodfacts' },
  { label: 'TACO', value: 'taco' },
  { label: 'USDA', value: 'usda' },
]

export function Foods() {
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  const [source, setSource] = useState('')
  const [selected, setSelected] = useState<string | null>(null)

  // Debounce the search input so we don't fire a request per keystroke.
  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])

  const searching = query.length > 0
  const search = useSearchFoods(query)
  const browse = useFoods(source)

  const isLoading = searching ? search.isLoading : browse.isLoading
  const foods = useMemo(
    () => (searching ? search.data : browse.data) ?? [],
    [searching, search.data, browse.data],
  )

  return (
    <div>
      <PageHeader eyebrow="Foods" title="Browser" />

      <div className="mb-5">
        <FrequentFoods />
      </div>

      <div className="relative mb-4">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          <SearchIcon width={18} height={18} />
        </span>
        <input
          value={rawQuery}
          onChange={(e) => setRawQuery(e.target.value)}
          placeholder="Search foods or aliases"
          aria-label="Search foods"
          className="w-full rounded-full border border-line bg-surface py-2.5 pl-10 pr-4 text-ink outline-none transition focus:border-primary"
        />
      </div>

      {!searching && (
        <div className="mb-6 flex flex-wrap gap-2">
          {SOURCES.map((s) => (
            <button
              key={s.value}
              onClick={() => setSource(s.value)}
              className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                source === s.value
                  ? 'border-transparent bg-primary-soft text-primary'
                  : 'border-line bg-surface text-muted hover:text-ink'
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>
      )}

      {isLoading ? (
        <Spinner label="Loading foods" />
      ) : !foods.length ? (
        <EmptyState
          icon={<FoodsIcon />}
          title={searching ? 'No matches' : 'No foods yet'}
          hint={
            searching
              ? 'Try a different name or alias.'
              : 'Foods appear here as you log meals. Turn on Demo mode to explore.'
          }
        />
      ) : (
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
      )}

      {selected && <FoodDetailModal foodID={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
