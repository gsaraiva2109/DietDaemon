#!/usr/bin/env node
// Generates dietbox-plan-4page.pdf: a small, hand-built, real 4-page PDF with
// a genuine text layer (no scanned images, no embedded fonts — just the
// standard Helvetica Type1 font, which needs no embedding for text-content
// extraction). Content is entirely synthetic/sanitized (no real names or
// professional identifiers) and mirrors a Dietbox-style export: 2 day types
// split across pages, a standalone substitution note, and a general plan
// note, each page carrying a unique greppable marker so tests can prove
// every page's content survives extraction in order (#224).
//
// Regenerate with: node src/routes/testdata/generate-dietbox-fixture.mjs
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const OUT = path.join(path.dirname(fileURLToPath(import.meta.url)), 'dietbox-plan-4page.pdf')

// Text-only, unaccented (ASCII) content: the standard Helvetica font's
// default encoding isn't guaranteed to round-trip accented Portuguese
// characters through getTextContent() without embedding a font, so the
// fixture sticks to plain ASCII rather than fighting PDF text encoding.
const PAGES = [
  [
    'DietBox Fixture Plan',
    'Dia de treino',
    'Refeicao: Cafe da manha - 07:00',
    'Opcao 1: 100g aveia em flocos, 1 banana media',
    'Nota unica pagina 1: TRK-P1-AVEIA',
  ],
  [
    'Dia de treino (continuacao)',
    'Refeicao: Almoco - 12:30',
    'Opcao 1: 150g peito de frango grelhado, 100g arroz integral',
    'Substituicao: trocar arroz integral por 150g batata doce',
    'Nota unica pagina 2: TRK-P2-FRANGO',
  ],
  [
    'Dia de descanso',
    'Refeicao: Cafe da manha - 08:00',
    'Opcao 1: 200ml iogurte natural, 30g granola',
    'Nota unica pagina 3: RST-P3-IOGURTE',
  ],
  [
    'Dia de descanso (continuacao)',
    'Refeicao: Jantar - 19:00',
    'Opcao 1: 120g salmao grelhado, salada verde a vontade',
    'Observacoes gerais: beber 2 litros de agua por dia',
    'Nota unica pagina 4: RST-P4-SALMAO',
  ],
]

function pdfEscape(str) {
  return str.replaceAll('\\', '\\\\').replaceAll('(', '\\(').replaceAll(')', '\\)')
}

function contentStreamBody(lines) {
  const ops = ['BT', '/F1 12 Tf', `72 720 Td`]
  lines.forEach((line, i) => {
    if (i > 0) ops.push('0 -16 Td')
    ops.push(`(${pdfEscape(line)}) Tj`)
  })
  ops.push('ET')
  return ops.join('\n')
}

// Object numbering: 1 catalog, 2 pages tree, then per page a (page, contents)
// pair (3/4, 5/6, 7/8, 9/10), and 11 the shared Helvetica font.
const FONT_OBJ = 11
const objects = new Map() // objNum -> body string (without "N 0 obj"/"endobj")

objects.set(1, '<< /Type /Catalog /Pages 2 0 R >>')

const pageObjNums = PAGES.map((_, i) => 3 + i * 2)
const kids = pageObjNums.map((n) => `${n} 0 R`).join(' ')
objects.set(2, `<< /Type /Pages /Kids [${kids}] /Count ${pageObjNums.length} >>`)

PAGES.forEach((lines, i) => {
  const pageObj = pageObjNums[i]
  const contentsObj = pageObj + 1
  objects.set(
    pageObj,
    `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 ${FONT_OBJ} 0 R >> >> /Contents ${contentsObj} 0 R >>`,
  )
  const body = contentStreamBody(lines)
  objects.set(contentsObj, `<< /Length ${Buffer.byteLength(body, 'latin1')} >>\nstream\n${body}\nendstream`)
})

objects.set(FONT_OBJ, '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>')

const maxObjNum = Math.max(...objects.keys())

let out = '%PDF-1.4\n'
const offsets = new Map()
for (let n = 1; n <= maxObjNum; n++) {
  const body = objects.get(n)
  if (!body) continue
  offsets.set(n, Buffer.byteLength(out, 'latin1'))
  out += `${n} 0 obj\n${body}\nendobj\n`
}

const xrefOffset = Buffer.byteLength(out, 'latin1')
out += `xref\n0 ${maxObjNum + 1}\n`
out += '0000000000 65535 f \n'
for (let n = 1; n <= maxObjNum; n++) {
  const offset = offsets.get(n)
  if (offset == null) {
    out += '0000000000 00000 f \n'
    continue
  }
  out += `${String(offset).padStart(10, '0')} 00000 n \n`
}
out += `trailer\n<< /Size ${maxObjNum + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`

writeFileSync(OUT, Buffer.from(out, 'latin1'))
console.log(`Wrote ${OUT} (${Buffer.byteLength(out, 'latin1')} bytes, ${PAGES.length} pages)`)
