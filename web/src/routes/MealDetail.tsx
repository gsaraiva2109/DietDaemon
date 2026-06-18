import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMeal, useDeleteItem } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Card, Pill, Spinner, Button } from '@/components/ui'
import { CorrectItemModal } from '@/components/CorrectItemModal'
import { SaveTemplateModal } from '@/components/SaveTemplateModal'
import { ShareCard } from '@/components/ShareCard'
import { ChevronLeft, LogIcon, CloseIcon, TemplateIcon, ShareIcon } from '@/components/icons'
import {
  clockTime,
  confidenceLabel,
  formatGrams,
  formatNumber,
  tierLabel,
} from '@/lib/format'
import { MACRO_KEYS, MACRO_META, type Macros } from '@/lib/types'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function MealDetail() {
  const { mealID } = useParams()
  const meal = useMeal(mealID)
  const del = useDeleteItem(mealID ?? '')
  const { demo } = useDemo()
  // null = closed; -1 = add mode; >=0 = correcting that index.
  const [editing, setEditing] = useState<number | null>(null)
  const [savingTemplate, setSavingTemplate] = useState(false)
  const [sharing, setSharing] = useState(false)

  if (meal.isLoading) return <Spinner label="Loading meal" />
  if (meal.isError || !meal.data) {
    return (
      <div>
        <BackLink />
        <p className="mt-6 text-muted">Meal not found.</p>
      </div>
    )
  }

  const m = meal.data
  const total = m.Items.reduce((s, it) => s + it.Macros.Calories, 0)
  const mealMacros = m.Items.reduce<Macros>(
    (s, it) => ({
      Calories: s.Calories + it.Macros.Calories,
      Protein: s.Protein + it.Macros.Protein,
      Carbs: s.Carbs + it.Macros.Carbs,
      Fat: s.Fat + it.Macros.Fat,
      Fiber: s.Fiber + it.Macros.Fiber,
    }),
    { ...ZERO },
  )

  return (
    <div>
      <BackLink />
      <PageHeader eyebrow={clockTime(m.At)} title={m.RawText || 'Logged meal'}>
        <div className="flex items-center gap-2">
          <Pill tone={m.ParserTier === 2 ? 'accent' : 'primary'}>{tierLabel(m.ParserTier)}</Pill>
          <Pill tone="muted">{confidenceLabel(m.Confidence)} confidence</Pill>
        </div>
      </PageHeader>

      <div className="mb-5 flex items-center justify-between gap-3">
        <p className="text-sm text-muted">
          <span className="text-2xl font-bold text-ink tnum">{formatNumber(total)}</span> kcal total
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="ghost" onClick={() => setSharing(true)} className="px-3 py-1.5 text-xs">
            <ShareIcon width={15} height={15} /> Share
          </Button>
          {!demo && (
            <>
              <Button
                variant="ghost"
                onClick={() => setSavingTemplate(true)}
                disabled={!m.Items.length}
                className="px-3 py-1.5 text-xs"
              >
                <TemplateIcon width={15} height={15} /> Save as template
              </Button>
              <Button variant="ghost" onClick={() => setEditing(-1)} className="px-3 py-1.5 text-xs">
                <LogIcon width={15} height={15} /> Add item
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="flex flex-col gap-3">
        {m.Items.map((it, i) => (
          <Card key={i} className="p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="font-semibold text-ink">{it.Match.Name || it.Parsed.RawPhrase}</p>
                <p className="mt-0.5 text-sm text-muted">
                  {formatGrams(it.Parsed.NormalizedGrams)} · {it.Match.Source}
                </p>
              </div>
              {!demo && (
                <div className="flex shrink-0 items-center gap-1">
                  <Button variant="ghost" onClick={() => setEditing(i)} className="px-3 py-1.5 text-xs">
                    Correct
                  </Button>
                  <button
                    onClick={() => del.mutate(i)}
                    disabled={del.isPending}
                    aria-label={`Remove ${it.Match.Name || it.Parsed.RawPhrase}`}
                    className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
                  >
                    <CloseIcon width={16} height={16} />
                  </button>
                </div>
              )}
            </div>
            <dl className="mt-3 grid grid-cols-5 gap-2 border-t border-line pt-3">
              {MACRO_KEYS.map((k) => (
                <div key={k}>
                  <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">{MACRO_META[k].label}</dt>
                  <dd className="font-semibold text-ink tnum">{Math.round(it.Macros[k])}</dd>
                </div>
              ))}
            </dl>
          </Card>
        ))}
        {!m.Items.length && <p className="text-muted">No items in this meal.</p>}
      </div>

      {editing !== null && (
        <CorrectItemModal
          meal={m}
          index={editing === -1 ? undefined : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {savingTemplate && (
        <SaveTemplateModal items={m.Items} onClose={() => setSavingTemplate(false)} />
      )}
      {sharing && (
        <ShareCard
          heading={m.RawText || 'Logged meal'}
          subtitle={clockTime(m.At)}
          consumed={mealMacros}
          onClose={() => setSharing(false)}
        />
      )}
    </div>
  )
}

function BackLink() {
  return (
    <Link to="/history" prefetch="intent" className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink">
      <ChevronLeft width={18} height={18} /> History
    </Link>
  )
}
