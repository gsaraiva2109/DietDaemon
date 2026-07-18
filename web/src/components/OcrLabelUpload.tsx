// "Scan label" trigger for the custom-food form. Uploads a photo, hands the
// extracted draft back to the caller for review/prefill, never touches any
// form state itself (issue #87: prefill-only, no auto-create).

import { useRef, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useOcrExtractCustomFood } from '@/lib/queries'
import type { NutritionLabelDraft } from '@/lib/types'
import { Button, FormError } from './ui'

export function OcrLabelUpload({ onExtracted }: { onExtracted: (draft: NutritionLabelDraft) => void }) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const scan = useOcrExtractCustomFood()

  function onSelect(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    scan.mutate(file, { onSuccess: onExtracted })
  }

  return (
    <div className="flex flex-col gap-1.5">
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        onChange={onSelect}
        className="hidden"
        aria-label={t('customFood.scanLabel')}
      />
      <Button
        type="button"
        variant="ghost"
        onClick={() => inputRef.current?.click()}
        disabled={scan.isPending}
        className="self-start"
      >
        {scan.isPending ? t('customFood.scanning') : t('customFood.scanLabel')}
      </Button>
      {scan.isError && <FormError>{t('customFood.scanError')}</FormError>}
    </div>
  )
}
