// Aliases, manage the alternate names that map free-text phrases to a food.
// A settings sub-page: search the library, then add/remove aliases per food.
// Write controls are disabled in demo mode (mirrors MealDetail).

import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useFoods, useSearchFoods, useAddAlias, useDeleteAlias } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, EmptyState, Spinner } from '@/components/ui'
import { ChevronLeft, CloseIcon, SearchIcon } from '@/components/icons'
import type { FoodDetail } from '@/lib/types'
import { stagger, fadeUp } from '@/lib/motion'

export function Aliases() {
  const { demo } = useDemo()
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')

  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])

  const searching = query.length > 0
  const search = useSearchFoods(query)
  const browse = useFoods('')

  const isLoading = searching ? search.isLoading : browse.isLoading
  const foods = ((searching ? search.data : browse.data) ?? []).slice(0, 30)

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Food aliases" />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Aliases are read only here.
        </p>
      )}

      <div className="relative mb-6">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          <SearchIcon width={18} height={18} />
        </span>
        <input
          value={rawQuery}
          onChange={(e) => setRawQuery(e.target.value)}
          placeholder="Search foods to edit aliases"
          aria-label="Search foods"
          className="w-full rounded-full border border-line bg-surface py-2.5 pl-10 pr-4 text-ink outline-none transition focus:border-primary"
        />
      </div>

      {isLoading ? (
        <Spinner label="Loading foods" />
      ) : !foods.length ? (
        <EmptyState title="No foods found" hint="Try a different search." />
      ) : (
        <motion.div
          variants={stagger}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-3"
        >
          {foods.map((f: FoodDetail) => (
            <motion.div key={f.food_id} variants={fadeUp}>
              <AliasRow food={f} demo={demo} />
            </motion.div>
          ))}
        </motion.div>
      )}
    </div>
  )
}

function AliasRow({ food, demo }: { food: FoodDetail; demo: boolean }) {
  const add = useAddAlias(food.food_id)
  const del = useDeleteAlias(food.food_id)
  const [value, setValue] = useState('')
  const aliases = food.aliases ?? []

  function submit() {
    const v = value.trim()
    if (!v || demo) return
    add.mutate(v)
    setValue('')
  }

  return (
    <Card className="p-4">
      <p className="font-semibold text-ink">{food.name}</p>

      <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
        {aliases.length === 0 && <span className="text-sm text-muted">No aliases yet.</span>}
        {aliases.map((a) => (
          <span
            key={a.alias}
            className="inline-flex items-center gap-1 rounded-full border border-line bg-surface-2 py-0.5 pl-2.5 pr-1 text-xs font-medium text-ink"
          >
            {a.alias}
            {!demo && (
              <button
                onClick={() => del.mutate(a.alias)}
                disabled={del.isPending}
                aria-label={`Remove alias ${a.alias}`}
                className="grid size-5 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
              >
                <CloseIcon width={12} height={12} />
              </button>
            )}
          </span>
        ))}
      </div>

      {!demo && (
        <div className="mt-3 flex gap-2">
          <input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') submit()
            }}
            placeholder="Add an alias"
            aria-label={`Add alias for ${food.name}`}
            className="min-w-0 flex-1 rounded-full border border-line bg-bg px-3.5 py-2 text-sm text-ink outline-none transition focus:border-primary"
          />
          <Button onClick={submit} disabled={!value.trim() || add.isPending} className="px-4 py-2 text-sm">
            Add
          </Button>
        </div>
      )}
    </Card>
  )
}
