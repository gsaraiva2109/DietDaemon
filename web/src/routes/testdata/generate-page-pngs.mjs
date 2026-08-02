#!/usr/bin/env node
// Generates page1.png..page4.png: tiny (4x4), solid-color PNGs with distinct
// bytes, for PlanImport.test.tsx's mocked-pdfToImages scanned-path fixtures
// (#224). No image library needed — a solid-color PNG is small enough to
// hand-assemble from its IHDR/IDAT/IEND chunks directly.
//
// Regenerate with: node src/routes/testdata/generate-page-pngs.mjs
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import zlib from 'node:zlib'

const DIR = path.dirname(fileURLToPath(import.meta.url))

function chunk(type, data) {
  const len = Buffer.alloc(4)
  len.writeUInt32BE(data.length, 0)
  const typeBuf = Buffer.from(type, 'ascii')
  const crc = Buffer.alloc(4)
  crc.writeUInt32BE(zlib.crc32(Buffer.concat([typeBuf, data])) >>> 0, 0)
  return Buffer.concat([len, typeBuf, data, crc])
}

function solidPng(size, [r, g, b]) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10])
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(size, 0)
  ihdr.writeUInt32BE(size, 4)
  ihdr[8] = 8 // bit depth
  ihdr[9] = 2 // color type: truecolor (RGB)
  const rowLength = 1 + size * 3 // filter-type byte + RGB per pixel
  const raw = Buffer.alloc(rowLength * size)
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const offset = y * rowLength + 1 + x * 3
      raw[offset] = r
      raw[offset + 1] = g
      raw[offset + 2] = b
    }
  }
  const idat = zlib.deflateSync(raw)
  return Buffer.concat([signature, chunk('IHDR', ihdr), chunk('IDAT', idat), chunk('IEND', Buffer.alloc(0))])
}

const PAGES = [
  { name: 'page1.png', color: [220, 60, 60] }, // red
  { name: 'page2.png', color: [60, 160, 90] }, // green
  { name: 'page3.png', color: [60, 110, 220] }, // blue
  { name: 'page4.png', color: [230, 190, 60] }, // yellow
]

for (const { name, color } of PAGES) {
  const buf = solidPng(4, color)
  writeFileSync(path.join(DIR, name), buf)
  console.log(`Wrote ${name} (${buf.length} bytes)`)
}
