// Runs pdfToText against a real, hand-built multi-page PDF through the real
// pdfjs-dist library (not a hand-rolled mock, unlike pdfToText.test.ts) to
// prove every source page's text actually reaches the combined payload, in
// order (#224 req. 2).
//
// jsdom (Vitest's default environment here) has no `Worker` global, and
// pdfjs-dist's browser build (what pdfjsLoader.ts imports for real browsers)
// can't run its fake-worker fallback for that case under Node/ESM — it tries
// to dynamically `import()` GlobalWorkerOptions.workerSrc, which Vite
// resolves to an unfetchable dev-server http(s) URL, and hits an internal
// message-handler bug besides. This is exactly what pdfjs-dist's own runtime
// warning ("Please use the `legacy` build in Node.js environments") is
// telling us, so this file remaps the `pdfjs-dist` specifier — scoped only to
// this test file, leaving pdfjsLoader.ts and every other test/production path
// untouched — to the real `pdfjs-dist/legacy/build/pdf.mjs`, which has the
// same public API and is still the real library, just the Node-friendly
// build of it.
import { describe, it, expect, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import path from 'node:path'

vi.mock('pdfjs-dist', async () => {
  // The legacy build's fake-worker path checks this global before falling
  // back to importing GlobalWorkerOptions.workerSrc itself (the same
  // unfetchable-URL problem as above), so pre-loading the worker module here
  // — via its plain, Node-resolvable package specifier — sidesteps it.
  ;(globalThis as unknown as { pdfjsWorker?: unknown }).pdfjsWorker = await import(
    'pdfjs-dist/legacy/build/pdf.worker.min.mjs'
  )
  return import('pdfjs-dist/legacy/build/pdf.mjs')
})

import { pdfToText } from './pdfToText'

// Without an explicit `standardFontDataUrl`, the legacy build can't load
// glyph-width metrics for a non-embedded standard font (Helvetica, as used
// by the fixture PDF) and warns once — harmless, since getTextContent()'s
// extracted strings don't depend on font metrics (confirmed by this test's
// own assertions below). vitest.setup.ts makes any console.warn fail a test
// by default, so this one specific, expected message is allowed through
// without weakening that guard for anything else.
function allowMissingStandardFontWarning() {
  vi.spyOn(console, 'warn').mockImplementation((...args) => {
    if (typeof args[0] === 'string' && args[0].includes('standardFontDataUrl')) return
    throw new Error(`Unexpected console.warn: ${args.map(String).join(' ')}`)
  })
}

const FIXTURE_PATH = path.join(import.meta.dirname, '../routes/testdata/dietbox-plan-4page.pdf')

const PAGE_MARKERS = ['TRK-P1-AVEIA', 'TRK-P2-FRANGO', 'RST-P3-IOGURTE', 'RST-P4-SALMAO']

function fixtureFile(): File {
  const bytes = readFileSync(FIXTURE_PATH)
  return new File([bytes], 'dietbox-plan-4page.pdf', { type: 'application/pdf' })
}

describe('pdfToText against a real 4-page PDF fixture', () => {
  it('extracts all 4 pages, in order, with each page carrying only its own content', async () => {
    allowMissingStandardFontWarning()
    const result = await pdfToText(fixtureFile())

    expect(result.status).toBe('ok')
    expect(result.pageCount).toBe(4)

    const markerPositions = PAGE_MARKERS.map((marker) => result.text.indexOf(marker))
    expect(markerPositions.every((pos) => pos !== -1)).toBe(true)
    expect(markerPositions).toEqual([...markerPositions].sort((a, b) => a - b))

    // Split into per-page segments on the `--- Page N ---` markers so each
    // fixture marker is asserted inside the section it actually belongs to,
    // not just "present somewhere in the joined string".
    const segments = result.text.split(/--- Page \d+ ---\n?/).filter(Boolean)
    expect(segments).toHaveLength(4)
    segments.forEach((segment, i) => {
      expect(segment).toContain(PAGE_MARKERS[i])
      PAGE_MARKERS.filter((_, j) => j !== i).forEach((otherMarker) => {
        expect(segment).not.toContain(otherMarker)
      })
    })

    // Content-level sanity: day types, meals, the standalone substitution
    // note, and the general plan note all made it through.
    expect(result.text).toContain('Dia de treino')
    expect(result.text).toContain('Dia de descanso')
    expect(result.text).toContain('Cafe da manha')
    expect(result.text).toContain('Almoco')
    expect(result.text).toContain('Jantar')
    expect(result.text).toContain('Substituicao: trocar arroz integral por 150g batata doce')
    expect(result.text).toContain('Observacoes gerais: beber 2 litros de agua por dia')
  })
})
