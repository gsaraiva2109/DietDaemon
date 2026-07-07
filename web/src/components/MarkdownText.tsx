// Assistant reply text rendered as markdown, styled against DietDaemon's own
// tokens rather than a typography plugin (no new dependency for what a
// handful of element overrides covers).

import { MarkdownTextPrimitive } from '@assistant-ui/react-markdown'

export function MarkdownText() {
  return (
    <MarkdownTextPrimitive
      smooth
      className="prose-chat"
      components={{
        p: (props) => <p className="mb-2.5 last:mb-0" {...props} />,
        strong: (props) => <strong className="font-semibold text-ink" {...props} />,
        a: (props) => (
          <a className="text-primary underline underline-offset-2" target="_blank" rel="noreferrer" {...props} />
        ),
        ul: (props) => <ul className="mb-2.5 list-disc space-y-1 pl-5 last:mb-0" {...props} />,
        ol: (props) => <ol className="mb-2.5 list-decimal space-y-1 pl-5 last:mb-0" {...props} />,
        li: (props) => <li className="text-ink" {...props} />,
        code: (props) => <code className="rounded bg-surface-2 px-1 py-0.5 text-[0.85em] text-ink" {...props} />,
        pre: (props) => (
          <pre className="mb-2.5 overflow-x-auto rounded-lg bg-surface-2 p-3 text-[0.85em] last:mb-0" {...props} />
        ),
        blockquote: (props) => (
          <blockquote className="mb-2.5 border-l-2 border-line pl-3 text-muted last:mb-0" {...props} />
        ),
        h1: (props) => <h2 className="mb-2 mt-1 text-base font-bold text-ink" {...props} />,
        h2: (props) => <h2 className="mb-2 mt-1 text-base font-bold text-ink" {...props} />,
        h3: (props) => <h3 className="mb-1.5 mt-1 text-sm font-bold text-ink" {...props} />,
      }}
    />
  )
}
