// Assistant reply text rendered as markdown, styled against DietDaemon's own
// tokens rather than a typography plugin (no new dependency for what a
// handful of element overrides covers).

import { MarkdownTextPrimitive } from '@assistant-ui/react-markdown'
import type { Components } from 'react-markdown'

// Overrides are hoisted to module scope so react-markdown gets stable
// component references instead of a new function per render.
const P: Components['p'] = (props) => <p className="mb-2.5 last:mb-0" {...props} />

const Strong: Components['strong'] = (props) => (
  <strong className="font-semibold text-ink" {...props} />
)

const Anchor: Components['a'] = (props) => (
  <a className="text-primary underline underline-offset-2" target="_blank" rel="noreferrer" {...props} />
)

const Ul: Components['ul'] = (props) => (
  <ul className="mb-2.5 list-disc space-y-1 pl-5 last:mb-0" {...props} />
)

const Ol: Components['ol'] = (props) => (
  <ol className="mb-2.5 list-decimal space-y-1 pl-5 last:mb-0" {...props} />
)

const Li: Components['li'] = (props) => <li className="text-ink" {...props} />

const Code: Components['code'] = (props) => (
  <code className="rounded bg-surface-2 px-1 py-0.5 text-[0.85em] text-ink" {...props} />
)

const Pre: Components['pre'] = (props) => (
  <pre className="mb-2.5 overflow-x-auto rounded-lg bg-surface-2 p-3 text-[0.85em] last:mb-0" {...props} />
)

const Blockquote: Components['blockquote'] = (props) => (
  <blockquote className="mb-2.5 border-l-2 border-line pl-3 text-muted last:mb-0" {...props} />
)

// Headings must spell out {children} explicitly (rather than relying on it
// riding along inside a spread) so the rendered element always has
// screen-reader-accessible content instead of looking empty.
const H1: Components['h1'] = ({ children, ...props }) => (
  <h1 className="mb-2 mt-1 text-base font-bold text-ink" {...props}>
    {children}
  </h1>
)

const H2: Components['h2'] = ({ children, ...props }) => (
  <h2 className="mb-2 mt-1 text-base font-bold text-ink" {...props}>
    {children}
  </h2>
)

const H3: Components['h3'] = ({ children, ...props }) => (
  <h3 className="mb-1.5 mt-1 text-sm font-bold text-ink" {...props}>
    {children}
  </h3>
)

const markdownComponents: Components = {
  p: P,
  strong: Strong,
  a: Anchor,
  ul: Ul,
  ol: Ol,
  li: Li,
  code: Code,
  pre: Pre,
  blockquote: Blockquote,
  h1: H1,
  h2: H2,
  h3: H3,
}

export function MarkdownText() {
  return <MarkdownTextPrimitive smooth className="prose-chat" components={markdownComponents} />
}
