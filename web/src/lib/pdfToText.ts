// Extracts a PDF's native text layer, client-side, page by page, joining
// pages into one string with `--- Page N ---` markers so the combined text
// can go through the existing paste-text extraction path (#193's
// /plans/extract/text) instead of the image-vision path — cheaper and more
// accurate whenever the PDF actually has a text layer worth reading (#220).
import type { PDFDocumentProxy } from 'pdfjs-dist'
import { loadPdfjs } from './pdfjsLoader'

// "Text effectively absent": below this many non-whitespace characters per
// page, native extraction isn't worth using — the caller should fall back to
// the scanned (image) path instead. A scanned-image-only PDF reports ~0
// chars/page; a real prescription page easily clears this even in a short
// one. Starting value, may need tuning after real-world use.
const MIN_CHARS_PER_PAGE = 20

// "Text likely malformed": a common signature of PDF font-encoding failures
// is a text layer that "extracts" but comes out as mostly single-character
// tokens (or replacement/control characters) instead of real words. Above
// this fraction of such tokens, treat the text as garbled rather than usable.
// Starting value, may need tuning after real-world use.
const MALFORMED_TOKEN_RATIO = 0.3

// Matches the Unicode replacement character or a C0 control character inside
// a token — both are common byproducts of a broken PDF font encoding.
// eslint-disable-next-line no-control-regex -- deliberately matching control chars as a garbling signal
const BAD_CHAR_RE = /[\uFFFD\x00-\x1f]/

export interface PdfTextResult {
  text: string
  pageCount: number
  status: 'ok' | 'empty' | 'malformed'
}

async function extractPageText(pdf: PDFDocumentProxy, pageNumber: number): Promise<string> {
  const page = await pdf.getPage(pageNumber)
  const content = await page.getTextContent()
  return content.items.map((item) => ('str' in item ? item.str : '')).join(' ')
}

function isMalformed(pageTexts: string[]): boolean {
  const tokens = pageTexts.join(' ').split(/\s+/).filter(Boolean)
  if (tokens.length === 0) return false
  const badTokens = tokens.filter((tok) => tok.length === 1 || BAD_CHAR_RE.test(tok)).length
  return badTokens / tokens.length > MALFORMED_TOKEN_RATIO
}

export async function pdfToText(file: File): Promise<PdfTextResult> {
  const pdfjs = await loadPdfjs()
  const data = await file.arrayBuffer()
  const pdf = await pdfjs.getDocument({ data }).promise
  const pageCount = pdf.numPages

  const pageTexts: string[] = []
  for (let pageNumber = 1; pageNumber <= pageCount; pageNumber++) {
    pageTexts.push(await extractPageText(pdf, pageNumber))
  }

  const nonWhitespaceChars = pageTexts.join('').replace(/\s/g, '').length
  const status: PdfTextResult['status'] =
    nonWhitespaceChars < MIN_CHARS_PER_PAGE * Math.max(pageCount, 1)
      ? 'empty'
      : isMalformed(pageTexts)
        ? 'malformed'
        : 'ok'

  const text = pageTexts.map((t, i) => `--- Page ${i + 1} ---\n${t}`).join('\n\n')
  return { text, pageCount, status }
}
