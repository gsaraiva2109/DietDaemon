import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import type { ComponentProps } from 'react'
import { MarkdownText } from './MarkdownText'

const SOURCE = vi.hoisted(
  () => `
# Heading One

## Heading Two

### Heading Three

A paragraph with **bold** text and a [link](https://example.com).

- item one
- item two

1. first
2. second

> a quote

Some \`inline code\`.

\`\`\`
block code
\`\`\`
`,
)

// `MarkdownTextPrimitive` reads its source text from assistant-ui's message
// context, which only exists inside a live thread runtime. The behavior this
// file owns is the `components` override map, so the primitive is swapped
// for plain `react-markdown` fed a fixed source string — that renders the
// same overrides against real markdown without standing up a whole runtime.
vi.mock('@assistant-ui/react-markdown', async () => {
  const { default: Markdown } = await import('react-markdown')
  return {
    MarkdownTextPrimitive: (props: ComponentProps<typeof Markdown> & { className?: string }) => {
      const { className, components } = props
      return (
        <div className={className}>
          <Markdown components={components}>{SOURCE}</Markdown>
        </div>
      )
    },
  }
})

describe('MarkdownText', () => {
  it('renders headings with accessible, non-empty content (S6850 regression)', () => {
    render(<MarkdownText />)

    expect(screen.getByRole('heading', { level: 1, name: 'Heading One' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 2, name: 'Heading Two' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 3, name: 'Heading Three' })).toBeInTheDocument()
  })

  it('renders paragraph, bold and link overrides', () => {
    render(<MarkdownText />)

    expect(screen.getByText('bold', { selector: 'strong' })).toBeInTheDocument()
    const link = screen.getByRole('link', { name: 'link' })
    expect(link).toHaveAttribute('href', 'https://example.com')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })

  it('renders unordered and ordered list overrides', () => {
    render(<MarkdownText />)

    expect(screen.getByText('item one').closest('ul')).toBeInTheDocument()
    expect(screen.getByText('first').closest('ol')).toBeInTheDocument()
  })

  it('renders blockquote and inline code overrides', () => {
    render(<MarkdownText />)

    expect(screen.getByText('a quote').closest('blockquote')).toBeInTheDocument()
    expect(screen.getByText('inline code', { selector: 'code' })).toBeInTheDocument()
  })

  it('renders fenced code blocks inside a pre override', () => {
    render(<MarkdownText />)

    expect(screen.getByText('block code').closest('pre')).toBeInTheDocument()
  })
})
