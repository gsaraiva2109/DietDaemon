// Photo-menu dining mode (#201): photograph a restaurant menu, an endpoint
// returns a flat list of dish candidates, the user picks one, optionally
// edits its name/description, and logs it. Unlike PlanImport.tsx's draft
// review, there's no catalog resolution step and no nested tree — the
// picked/edited text goes straight to the free-text meal parser server-side,
// which forces low confidence (it's a rough restaurant estimate, always).

import { useState, type SyntheticEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useExtractMenuFromImage, useLogMenuDish } from '@/lib/queries'
import { Card, Button, Pill } from '@/components/ui'
import type { MenuDishCandidate } from '@/lib/types'

type Stage =
  | { kind: 'upload' }
  | { kind: 'error'; message: string }
  | { kind: 'unreadable' }
  | { kind: 'picking'; dishes: MenuDishCandidate[] }
  | { kind: 'editing'; dish: MenuDishCandidate }

export function MenuDiningCard() {
  const { t } = useTranslation()
  const [stage, setStage] = useState<Stage>({ kind: 'upload' })
  const [justLogged, setJustLogged] = useState(false)
  const extract = useExtractMenuFromImage()

  function handleFile(file: File) {
    setJustLogged(false)
    extract.mutate(file, {
      onSuccess: (draft) => {
        if (draft.unreadable) {
          setStage({ kind: 'unreadable' })
        } else {
          setStage({ kind: 'picking', dishes: draft.dishes })
        }
      },
      onError: (err) => setStage({ kind: 'error', message: err instanceof Error ? err.message : t('menu.extractFailed') }),
    })
  }

  function onDishLogged() {
    setStage({ kind: 'upload' })
    setJustLogged(true)
  }

  if (stage.kind === 'error') {
    return (
      <Card className="p-5">
        <p className="mb-3 text-sm font-medium text-accent" role="alert">
          {stage.message}
        </p>
        <Button variant="ghost" onClick={() => setStage({ kind: 'upload' })} className="px-3 py-1.5 text-xs">
          {t('menu.tryAgain')}
        </Button>
      </Card>
    )
  }

  if (stage.kind === 'unreadable') {
    return (
      <Card className="p-5">
        <p className="mb-3 text-sm font-medium text-accent" role="alert">
          {t('menu.unreadable')}
        </p>
        <Button variant="ghost" onClick={() => setStage({ kind: 'upload' })} className="px-3 py-1.5 text-xs">
          {t('menu.tryAgain')}
        </Button>
      </Card>
    )
  }

  if (stage.kind === 'picking') {
    return (
      <DishPicker
        dishes={stage.dishes}
        onPick={(dish) => setStage({ kind: 'editing', dish })}
        onCancel={() => setStage({ kind: 'upload' })}
      />
    )
  }

  if (stage.kind === 'editing') {
    return <DishEditor dish={stage.dish} onBack={() => setStage({ kind: 'picking', dishes: [stage.dish] })} onLogged={onDishLogged} />
  }

  return (
    <Card className="p-5">
      <p className="mb-1 font-semibold text-ink">{t('menu.uploadTitle')}</p>
      <p className="mb-3 text-sm text-muted">{t('menu.uploadHint')}</p>
      <label htmlFor="menu-import-photo" className="mb-1.5 block text-sm font-medium text-ink">
        {t('menu.choosePhotoFile')}
      </label>
      <input
        id="menu-import-photo"
        type="file"
        accept="image/jpeg,image/png"
        disabled={extract.isPending}
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) handleFile(file)
        }}
        className="block w-full text-sm text-ink"
      />
      {extract.isPending && <p className="mt-2 text-sm text-muted">{t('menu.extracting')}</p>}
      {justLogged && <p className="mt-3 text-sm font-medium text-primary">{t('menu.loggedSuccess')}</p>}
    </Card>
  )
}

function DishPicker({
  dishes,
  onPick,
  onCancel,
}: Readonly<{
  dishes: MenuDishCandidate[]
  onPick: (dish: MenuDishCandidate) => void
  onCancel: () => void
}>) {
  const { t } = useTranslation()
  return (
    <Card className="p-5">
      <p className="mb-3 font-semibold text-ink">{t('menu.pickDish')}</p>
      {dishes.length ? (
        <ul className="flex flex-col gap-2">
          {dishes.map((dish, i) => (
            <li key={`${dish.name}-${i}`}>
              <button
                type="button"
                onClick={() => onPick(dish)}
                className="w-full rounded-lg border border-line bg-surface px-3 py-2 text-left transition hover:border-primary"
              >
                <p className="text-sm font-medium text-ink">{dish.name}</p>
                {dish.description && <p className="text-xs text-muted">{dish.description}</p>}
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted">{t('menu.noDishesFound')}</p>
      )}
      <div className="mt-4 flex justify-end">
        <Button variant="ghost" onClick={onCancel} className="px-3 py-1.5 text-sm">
          {t('menu.cancel')}
        </Button>
      </div>
    </Card>
  )
}

function DishEditor({
  dish,
  onBack,
  onLogged,
}: Readonly<{
  dish: MenuDishCandidate
  onBack: () => void
  onLogged: () => void
}>) {
  const { t } = useTranslation()
  const [name, setName] = useState(dish.name)
  const [description, setDescription] = useState(dish.description)
  const logDish = useLogMenuDish()

  function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!name.trim() || logDish.isPending) return
    logDish.mutate({ name: name.trim(), description: description.trim() }, { onSuccess: onLogged })
  }

  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center justify-between gap-2">
        <p className="font-semibold text-ink">{t('menu.pickDish')}</p>
        <Pill tone="accent">{t('menu.lowConfidenceBadge')}</Pill>
      </div>
      <form onSubmit={onSubmit} className="flex flex-col gap-3">
        <label htmlFor="menu-dish-name" className="block text-sm font-medium text-ink">
          {t('menu.nameLabel')}
        </label>
        <input
          id="menu-dish-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-sm text-ink outline-none focus:border-primary"
        />
        <label htmlFor="menu-dish-description" className="block text-sm font-medium text-ink">
          {t('menu.descriptionLabel')}
        </label>
        <textarea
          id="menu-dish-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="w-full resize-none rounded-lg border border-line bg-bg px-3 py-2 text-sm text-ink outline-none focus:border-primary"
        />
        {logDish.isError && (
          <p className="text-sm font-medium text-accent" role="alert">
            {logDish.error instanceof Error ? logDish.error.message : t('menu.logFailed')}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onBack} className="px-3 py-1.5 text-sm">
            {t('menu.back')}
          </Button>
          <Button type="submit" disabled={!name.trim() || logDish.isPending} className="px-3 py-1.5 text-sm">
            {logDish.isPending ? t('menu.logging') : t('menu.logDish')}
          </Button>
        </div>
      </form>
    </Card>
  )
}
