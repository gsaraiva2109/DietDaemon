// Full-bleed AI chat assistant. Natural-language front end to anything a
// slash-command already does (log a meal, /suggest, /status, ...) plus
// free-form diet questions — additive, the bots/commands themselves are
// untouched. Built on assistant-ui's useLocalRuntime with a custom
// ChatModelAdapter (lib/chatRuntime.ts) that speaks our own SSE wire format.

import { useEffect, useMemo, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  AssistantRuntimeProvider,
  ThreadPrimitive,
  ComposerPrimitive,
  MessagePrimitive,
  ActionBarPrimitive,
  useLocalRuntime,
  useThreadRuntime,
  type ThreadMessageLike,
  AuiIf,
  groupPartByType,
} from '@assistant-ui/react';
import {
  useChatSessions,
  useCreateChatSession,
  useChatMessages,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { createChatModelAdapter } from '@/lib/chatRuntime'
import { PageHeader } from '@/components/PageHeader'
import { Button, EmptyState, Spinner } from '@/components/ui'
import { MarkdownText } from '@/components/MarkdownText'
import { ToolCallChip } from '@/components/ToolCallChip'
import { ToolCallGroup } from '@/components/ToolCallGroup'
import { ChatIcon, SendIcon, LogIcon, CopyIcon, CheckIcon, ChevronDown } from '@/components/icons'
import { fadeUp, stagger, easeOut } from '@/lib/motion'
import type { ChatMessageRecord, ChatSession } from '@/lib/types'

const EXAMPLES = [
  "I skipped breakfast, what should I eat?",
  'Log 200g grilled chicken and a banana',
  'How much protein do I have left today?',
]

function toInitialMessages(records: ChatMessageRecord[]): ThreadMessageLike[] {
  return records
    .filter((r): r is ChatMessageRecord & { role: 'user' | 'assistant' } => r.role === 'user' || r.role === 'assistant')
    .map((r) => ({ role: r.role, content: r.content }))
}

export function Chat() {
  const { demo } = useDemo()
  const sessions = useChatSessions()
  const createSession = useCreateChatSession()
  const [sessionID, setSessionID] = useState<string | null>(null)
  const [railOpen, setRailOpen] = useState(false)

  const sortedSessions = useMemo(
    () => [...(sessions.data ?? [])].sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
    [sessions.data],
  )

  // Land on the most recently updated session unless the user picked one
  // explicitly; derived at render time, no effect needed for this part.
  const activeSessionID = sessionID ?? sortedSessions[0]?.id ?? null

  // A brand-new user has no sessions yet: spin one up. This is a genuine
  // external side effect (a POST), so it's the one thing that does belong in
  // an effect — the resulting setState happens inside the mutation callback,
  // not synchronously in the effect body.
  //
  // Guard is a ref, not createSession.isPending: StrictMode double-invokes
  // this effect synchronously before the first mutate() call re-renders, so
  // an isPending-based guard hasn't flipped yet on the second pass and both
  // fire, creating two sessions.
  const creatingSession = useRef(false)
  useEffect(() => {
    if (demo || !sessions.isSuccess || sortedSessions.length > 0 || creatingSession.current) return
    creatingSession.current = true
    createSession.mutate(undefined, {
      onSuccess: (s) => setSessionID(s.id),
      onSettled: () => { creatingSession.current = false },
    })
  }, [demo, sessions.isSuccess, sortedSessions.length, createSession])

  const messages = useChatMessages(activeSessionID)

  if (demo) {
    return (
      <div>
        <PageHeader eyebrow="Assistant" title="Chat" />
        <EmptyState
          icon={<ChatIcon width={28} height={28} />}
          title="Chat needs a real account"
          hint="This talks to a live AI provider, so sample data can't stand in for it. Turn off demo mode to try it."
        />
      </div>
    )
  }

  if (sessions.isError) {
    return (
      <div>
        <PageHeader eyebrow="Assistant" title="Chat" />
        <EmptyState
          icon={<ChatIcon width={28} height={28} />}
          title="Assistant unavailable"
          hint={sessions.error instanceof Error ? sessions.error.message : 'Could not reach the chat backend.'}
        />
      </div>
    )
  }

  function newChat() {
    createSession.mutate(undefined, { onSuccess: (s) => setSessionID(s.id) })
    setRailOpen(false)
  }

  return (
    <div className="flex h-[calc(100dvh-8.5rem)] flex-col md:h-[calc(100dvh-7rem)]">
      <PageHeader eyebrow="Assistant" title="Chat">
        <Button variant="ghost" className="px-3 py-1.5 text-xs md:hidden" onClick={() => setRailOpen((o) => !o)}>
          History
        </Button>
      </PageHeader>

      <div className="relative flex min-h-0 flex-1 gap-4">
        <SessionRail
          sessions={sortedSessions}
          activeID={activeSessionID}
          loading={sessions.isLoading}
          open={railOpen}
          onClose={() => setRailOpen(false)}
          onSelect={(id) => {
            setSessionID(id)
            setRailOpen(false)
          }}
          onNew={newChat}
        />

        <div className="flex flex-1 flex-col overflow-hidden rounded-xl border border-line bg-surface">
          {!activeSessionID || messages.isLoading ? (
            <div className="grid flex-1 place-items-center">
              <Spinner label="Loading conversation" />
            </div>
          ) : (
            <ChatThread
              key={activeSessionID}
              sessionID={activeSessionID}
              initial={toInitialMessages(messages.data ?? [])}
            />
          )}
        </div>
      </div>
    </div>
  )
}

// --- Session rail: desktop-persistent, mobile-overlay --------------------

function SessionRail({
  sessions,
  activeID,
  loading,
  open,
  onClose,
  onSelect,
  onNew,
}: {
  sessions: ChatSession[]
  activeID: string | null
  loading: boolean
  open: boolean
  onClose: () => void
  onSelect: (id: string) => void
  onNew: () => void
}) {
  const list = (
    <div className="flex h-full w-64 flex-col gap-1 overflow-y-auto rounded-xl border border-line bg-surface p-2">
      <button
        onClick={onNew}
        className="mb-1 flex items-center gap-2 rounded-lg px-3 py-2.5 text-sm font-semibold text-primary transition hover:bg-primary-soft"
      >
        <LogIcon width={16} height={16} /> New chat
      </button>
      {loading && <Spinner label="Loading chats" />}
      {!loading && sessions.length === 0 && (
        <p className="px-3 py-2 text-xs text-muted">No conversations yet.</p>
      )}
      {sessions.map((s) => (
        <button
          key={s.id}
          onClick={() => onSelect(s.id)}
          className={`truncate rounded-lg px-3 py-2 text-left text-sm transition ${
            s.id === activeID ? 'bg-primary-soft text-primary' : 'text-muted hover:bg-surface-2 hover:text-ink'
          }`}
        >
          {s.title || 'New conversation'}
        </button>
      ))}
    </div>
  )

  return (
    <>
      <div className="hidden shrink-0 md:block">{list}</div>
      <AnimatePresence>
        {open && (
          <motion.div
            className="fixed inset-0 md:hidden"
            style={{ zIndex: 1200 }}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <div className="absolute inset-0 bg-ink/30 backdrop-blur-sm" onClick={onClose} />
            <motion.div
              className="absolute inset-y-0 left-0 w-72 max-w-[80vw] p-2"
              initial={{ x: '-100%' }}
              animate={{ x: 0 }}
              exit={{ x: '-100%' }}
              transition={{ duration: 0.32, ease: easeOut }}
            >
              {list}
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  )
}

// --- Thread: one runtime per session, remounted (key=sessionID) on switch --

function ChatThread({ sessionID, initial }: { sessionID: string; initial: ThreadMessageLike[] }) {
  const adapter = useMemo(() => createChatModelAdapter(() => sessionID), [sessionID])
  const runtime = useLocalRuntime(adapter, { initialMessages: initial })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <ThreadPrimitive.Root className="flex flex-1 flex-col overflow-hidden">
        <ThreadPrimitive.Viewport className="flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          <AuiIf condition={(s) => s.thread.isEmpty}>
            <ChatEmptyState />
          </AuiIf>
          <ThreadPrimitive.Messages components={{ UserMessage, AssistantMessage }} />
        </ThreadPrimitive.Viewport>

        <div className="relative">
          <ThreadPrimitive.ScrollToBottom asChild>
            <button
              aria-label="Scroll to latest"
              className="absolute -top-11 right-4 grid size-8 place-items-center rounded-full border border-line bg-surface text-muted shadow-soft transition hover:text-ink disabled:hidden"
            >
              <ChevronDown width={16} height={16} />
            </button>
          </ThreadPrimitive.ScrollToBottom>
          <Composer />
        </div>
      </ThreadPrimitive.Root>
    </AssistantRuntimeProvider>
  );
}

function ChatEmptyState() {
  const runtime = useThreadRuntime()
  return (
    <motion.div
      variants={stagger}
      initial="hidden"
      animate="show"
      className="grid h-full place-items-center text-center"
    >
      <div className="max-w-md">
        <motion.p variants={fadeUp} className="text-lg font-semibold text-ink">
          What do you need?
        </motion.p>
        <motion.p variants={fadeUp} className="mt-1.5 text-sm text-muted">
          Ask for a meal suggestion, log something, or check where you stand today.
        </motion.p>
        <motion.div variants={fadeUp} className="mt-5 flex flex-wrap justify-center gap-2">
          {EXAMPLES.map((ex) => (
            <button
              key={ex}
              type="button"
              onClick={() => runtime.append(ex)}
              className="rounded-full border border-line bg-surface-2 px-3 py-1.5 text-sm text-muted transition hover:text-ink"
            >
              {ex}
            </button>
          ))}
        </motion.div>
      </div>
    </motion.div>
  )
}

function UserMessage() {
  return (
    <MessagePrimitive.Root className="mb-4 flex justify-end">
      <div className="max-w-[80%] rounded-2xl bg-primary px-4 py-2.5 text-sm text-primary-ink">
        <MessagePrimitive.Parts />
      </div>
    </MessagePrimitive.Root>
  );
}

// Coalesces adjacent tool-call parts into a single "group-tools" node so a
// multi-step run (search, search again, log...) renders as one collapsible
// panel instead of a stack of individually-boxed calls.
const toolGroupBy = groupPartByType({ 'tool-call': ['group-tools'] })

function AssistantMessage() {
  return (
    <MessagePrimitive.Root className="group mb-4 flex flex-col items-start">
      <div className="max-w-[85%] rounded-2xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-ink">
        <MessagePrimitive.GroupedParts groupBy={toolGroupBy}>
          {({ part, children }) => {
            switch (part.type) {
              case 'group-tools':
                return (
                  <ToolCallGroup running={part.status.type === 'running'} count={part.indices.length}>
                    {children}
                  </ToolCallGroup>
                )
              case 'text':
                return <MarkdownText />
              case 'tool-call':
                return <ToolCallChip {...part} />
              default:
                return null
            }
          }}
        </MessagePrimitive.GroupedParts>
      </div>
      <ActionBarPrimitive.Root
        autohide="not-last"
        className="mt-1 flex gap-2 pl-1 text-muted opacity-0 transition group-hover:opacity-100"
      >
        <ActionBarPrimitive.Copy className="hover:text-ink" aria-label="Copy reply">
          <AuiIf condition={(s) => s.message.isCopied}>
            <CheckIcon width={13} height={13} />
          </AuiIf>
          <AuiIf condition={(s) => !s.message.isCopied}>
            <CopyIcon width={13} height={13} />
          </AuiIf>
        </ActionBarPrimitive.Copy>
      </ActionBarPrimitive.Root>
    </MessagePrimitive.Root>
  );
}

function Composer() {
  return (
    <ComposerPrimitive.Root className="flex items-end gap-2 border-t border-line bg-surface p-3">
      <ComposerPrimitive.Input
        rows={1}
        autoFocus
        placeholder="Ask anything, or tell me what you ate…"
        className="max-h-40 flex-1 resize-none bg-transparent px-2 py-2 text-sm text-ink outline-none placeholder:text-muted/70"
      />
      <AuiIf condition={(s) => !s.thread.isRunning}>
        <ComposerPrimitive.Send asChild>
          <button
            aria-label="Send"
            className="grid size-9 shrink-0 place-items-center rounded-full bg-primary text-primary-ink transition hover:brightness-105 disabled:opacity-40"
          >
            <SendIcon width={16} height={16} />
          </button>
        </ComposerPrimitive.Send>
      </AuiIf>
      <AuiIf condition={(s) => s.thread.isRunning}>
        <ComposerPrimitive.Cancel asChild>
          <button
            aria-label="Stop"
            className="grid size-9 shrink-0 place-items-center rounded-full border border-line bg-surface text-ink transition hover:bg-surface-2"
          >
            <span className="size-3 rounded-sm bg-ink" />
          </button>
        </ComposerPrimitive.Cancel>
      </AuiIf>
    </ComposerPrimitive.Root>
  );
}
