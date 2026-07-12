// Full-bleed AI chat assistant. Natural-language front end to anything a
// slash-command already does (log a meal, /suggest, /status, ...) plus
// free-form diet questions — additive, the bots/commands themselves are
// untouched.
//
// Built on assistant-ui's RemoteThreadListAdapter (lib/chatThreadListAdapter.ts)
// so the sidebar is a real ThreadListPrimitive backed by our own session
// backend (list/create/soft-delete/restore), not hand-rolled state. Each
// active thread gets its own useLocalRuntime with a custom ChatModelAdapter
// (lib/chatRuntime.ts) that speaks our own SSE wire format.

import { Suspense, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useQuery, useSuspenseQuery } from '@tanstack/react-query'
import {
  AssistantRuntimeProvider,
  ThreadPrimitive,
  ThreadListPrimitive,
  ThreadListItemPrimitive,
  ComposerPrimitive,
  MessagePrimitive,
  ActionBarPrimitive,
  ErrorPrimitive,
  BranchPickerPrimitive,
  useLocalRuntime,
  useRemoteThreadListRuntime,
  useThreadListItemRuntime,
  useMessageTiming,
  useAui,
  useAuiState,
  type ThreadMessageLike,
  type ToolCallMessagePartProps,
  AuiIf,
  groupPartByType,
} from '@assistant-ui/react'
import { api } from '@/lib/api'
import { useDemo } from '@/lib/demo'
import { createChatAdapters } from '@/lib/chatRuntime'
import { chatThreadListAdapter } from '@/lib/chatThreadListAdapter'
import { PageHeader } from '@/components/PageHeader'
import { Button, EmptyState, Spinner } from '@/components/ui'
import { MarkdownText } from '@/components/MarkdownText'
import { ToolCallChip } from '@/components/ToolCallChip'
import { ToolCallGroup } from '@/components/ToolCallGroup'
import { DeleteChatSessionModal } from '@/components/DeleteChatSessionModal'
import {
  ChatIcon,
  SendIcon,
  LogIcon,
  CopyIcon,
  CheckIcon,
  ChevronDown,
  TrashIcon,
  RefreshIcon,
} from '@/components/icons'
import { fadeUp, stagger, easeOut } from '@/lib/motion'
import type { ChatMessageRecord } from '@/lib/types'

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
  const [railOpen, setRailOpen] = useState(false)
  // Distinct from the thread-list runtime's own (silent) list() failures —
  // this surfaces a real "assistant unavailable" screen instead of a quietly
  // empty sidebar when the backend can't be reached at all.
  const health = useQuery({ queryKey: ['chat', 'health'], queryFn: api.chat.listSessions, retry: 1 })

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

  if (health.isError) {
    return (
      <div>
        <PageHeader eyebrow="Assistant" title="Chat" />
        <EmptyState
          icon={<ChatIcon width={28} height={28} />}
          title="Assistant unavailable"
          hint={health.error instanceof Error ? health.error.message : 'Could not reach the chat backend.'}
        />
      </div>
    )
  }

  return (
    <div className="flex h-[calc(100dvh-8.5rem)] flex-col md:h-[calc(100dvh-7rem)]">
      <PageHeader eyebrow="Assistant" title="Chat">
        <Button variant="ghost" className="px-3 py-1.5 text-xs md:hidden" onClick={() => setRailOpen((o) => !o)}>
          History
        </Button>
      </PageHeader>

      <Suspense
        fallback={
          <div className="grid flex-1 place-items-center">
            <Spinner label="Loading conversation" />
          </div>
        }
      >
        <ChatApp railOpen={railOpen} onCloseRail={() => setRailOpen(false)} />
      </Suspense>
    </div>
  )
}

// Constructs the remote-thread-list runtime and everything downstream of it.
// Split out from Chat() (rather than called there directly) because
// useRemoteThreadListRuntime's runtimeHook (useChatThreadRuntime) runs its own
// useSuspenseQuery before this component returns any JSX — if it suspends
// (e.g. switching to a different conversation resuspends on that thread's
// remoteId), the nearest boundary that can catch it is the one wrapping
// *this* component, not one further down in the JSX Chat() itself returns.
// Catching it here, below Chat()'s own render, keeps that resuspend from ever
// reaching the app-level route-transition AnimatePresence in App.tsx, whose
// interrupted opacity animation was otherwise getting stuck at 0.
function ChatApp({ railOpen, onCloseRail }: { railOpen: boolean; onCloseRail: () => void }) {
  const runtime = useRemoteThreadListRuntime({
    runtimeHook: useChatThreadRuntime,
    adapter: chatThreadListAdapter,
  })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <div className="relative flex min-h-0 flex-1 gap-4">
        <SessionRail open={railOpen} onClose={onCloseRail} />

        <div className="flex flex-1 flex-col overflow-hidden rounded-xl border border-line bg-surface">
          <Suspense
            fallback={
              <div className="grid flex-1 place-items-center">
                <Spinner label="Loading conversation" />
              </div>
            }
          >
            <ChatThread />
          </Suspense>
        </div>
      </div>
    </AssistantRuntimeProvider>
  )
}

// Per-thread runtime: a fresh instance per distinct local thread id (new or
// existing). `remoteId` is undefined until the thread's first message
// triggers chatThreadListAdapter.initialize() — the ChatModelAdapter reads it
// lazily at send-time via getSessionID, so this instance doesn't need to
// remount when that happens.
function useChatThreadRuntime() {
  const itemRuntime = useThreadListItemRuntime()
  const remoteId = itemRuntime.getState().remoteId ?? null

  const { data } = useSuspenseQuery({
    queryKey: ['chat', 'messages', remoteId],
    queryFn: () => (remoteId ? api.chat.getMessages(remoteId) : Promise.resolve([] as ChatMessageRecord[])),
  })
  const initialMessages = useMemo(() => toInitialMessages(data), [data])

  const adapters = useMemo(
    () => createChatAdapters(() => itemRuntime.getState().remoteId ?? null),
    [itemRuntime],
  )

  return useLocalRuntime(adapters.modelAdapter, {
    initialMessages: data.length > 0 ? initialMessages : undefined,
    adapters: { suggestion: adapters.suggestionAdapter },
  })
}

// --- Session rail: desktop-persistent, mobile-overlay --------------------

function SessionRail({ open, onClose }: { open: boolean; onClose: () => void }) {
  const isLoading = useAuiState((s) => s.threads.isLoading)
  const count = useAuiState((s) => s.threads.threadIds.length)

  const list = (
    <div className="flex h-full w-64 flex-col gap-1 overflow-y-auto rounded-xl border border-line bg-surface p-2">
      <ThreadListPrimitive.New asChild>
        <button
          onClick={onClose}
          className="mb-1 flex items-center gap-2 rounded-lg px-3 py-2.5 text-sm font-semibold text-primary transition hover:bg-primary-soft"
        >
          <LogIcon width={16} height={16} /> New chat
        </button>
      </ThreadListPrimitive.New>
      {isLoading && <Spinner label="Loading chats" />}
      {!isLoading && count === 0 && <p className="px-3 py-2 text-xs text-muted">No conversations yet.</p>}
      <ThreadListPrimitive.Items>{() => <SessionRow onSelect={onClose} />}</ThreadListPrimitive.Items>
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

function SessionRow({ onSelect }: { onSelect: () => void }) {
  const aui = useAui()
  const [confirming, setConfirming] = useState(false)

  return (
    <ThreadListItemPrimitive.Root className="group relative flex items-center rounded-lg data-active:bg-primary-soft data-active:text-primary">
      <ThreadListItemPrimitive.Trigger
        onClick={onSelect}
        className="min-w-0 flex-1 truncate rounded-lg px-3 py-2 text-left text-sm text-muted transition group-data-active:text-primary hover:bg-surface-2 hover:text-ink group-data-active:hover:bg-transparent"
      >
        <ThreadListItemPrimitive.Title fallback="New conversation" />
      </ThreadListItemPrimitive.Trigger>
      <button
        type="button"
        aria-label="Delete conversation"
        onClick={() => setConfirming(true)}
        className="mr-1 shrink-0 rounded p-1.5 text-muted opacity-0 transition hover:text-ink group-hover:opacity-100"
      >
        <TrashIcon width={14} height={14} />
      </button>
      {confirming && (
        <DeleteChatSessionModal
          onCancel={() => setConfirming(false)}
          onConfirm={() => {
            aui.threadListItem().archive()
            setConfirming(false)
          }}
        />
      )}
    </ThreadListItemPrimitive.Root>
  )
}

// --- Thread ----------------------------------------------------------------

function ChatThread() {
  return (
    <ThreadPrimitive.Root className="flex flex-1 flex-col overflow-hidden">
      <ThreadPrimitive.Viewport className="flex-1 overflow-y-auto px-4 py-5 sm:px-6">
        <AuiIf condition={(s) => s.thread.isEmpty}>
          <ChatEmptyState />
        </AuiIf>
        <ThreadPrimitive.Messages>
          {({ message }) => (message.role === 'user' ? <UserMessage /> : <AssistantMessage />)}
        </ThreadPrimitive.Messages>
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
        <Suggestions />
        <Composer />
      </div>
    </ThreadPrimitive.Root>
  )
}

function ChatEmptyState() {
  const aui = useAui()
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
              onClick={() => aui.thread().append(ex)}
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

// Backend-driven quick replies (#55): rendered when the model's last turn
// ended with a suggestions block (lib/chatRuntime.ts's SuggestionAdapter).
// Same chip look as the empty-state examples above. Plain buttons rather than
// SuggestionPrimitive — that primitive binds to a separate static-suggestions
// store scope in this assistant-ui version, not `thread.suggestions` (the one
// SuggestionAdapter.generate() populates), so there's no ready-made trigger
// for this particular data.
function Suggestions() {
  const suggestions = useAuiState((s) => s.thread.suggestions)
  const aui = useAui()
  if (!suggestions.length) return null

  return (
    <div className="flex flex-wrap gap-2 border-t border-line bg-surface px-3 pt-3">
      {suggestions.map((s, i) => (
        <button
          key={i}
          type="button"
          onClick={() => aui.thread().append(s.prompt)}
          className="rounded-full border border-line bg-surface-2 px-3 py-1.5 text-sm text-muted transition hover:text-ink"
        >
          {s.prompt}
        </button>
      ))}
    </div>
  )
}

function UserMessage() {
  return (
    <MessagePrimitive.Root className="group mb-4 flex flex-col items-end">
      <div className="max-w-[80%] rounded-2xl bg-primary px-4 py-2.5 text-sm text-primary-ink">
        <MessagePrimitive.Parts />
      </div>
      <BranchPickerPrimitive.Root
        hideWhenSingleBranch
        className="mt-1 flex items-center gap-1 pr-1 text-muted opacity-0 transition group-hover:opacity-100"
      >
        <BranchPickerPrimitive.Previous asChild>
          <button aria-label="Previous version" className="hover:text-ink">
            <ChevronDown width={12} height={12} className="rotate-90" />
          </button>
        </BranchPickerPrimitive.Previous>
        <span className="text-[11px] tnum">
          <BranchPickerPrimitive.Number />/<BranchPickerPrimitive.Count />
        </span>
        <BranchPickerPrimitive.Next asChild>
          <button aria-label="Next version" className="hover:text-ink">
            <ChevronDown width={12} height={12} className="-rotate-90" />
          </button>
        </BranchPickerPrimitive.Next>
      </BranchPickerPrimitive.Root>
    </MessagePrimitive.Root>
  );
}

// Coalesces adjacent tool-call parts into a single "group-tools" node so a
// multi-step run (search, search again, log...) renders as one collapsible
// panel instead of a stack of individually-boxed calls.
const toolGroupBy = groupPartByType({ 'tool-call': ['group-tools'] })

function AssistantMessage() {
  const status = useAuiState((s) => s.message.status)
  const hasError = status?.type === 'incomplete' && status.reason === 'error'
  const timing = useMessageTiming()

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
                return part.toolName === 'logmeal' ? (
                  <LogMealToolCard {...(part as unknown as ToolCallMessagePartProps)} />
                ) : (
                  <ToolCallChip {...part} />
                )
              default:
                return null
            }
          }}
        </MessagePrimitive.GroupedParts>
      </div>

      {hasError && (
        <ErrorPrimitive.Root className="mt-1.5 max-w-[85%] rounded-xl border border-accent/30 bg-accent/10 px-4 py-2.5 text-sm text-accent">
          <ErrorPrimitive.Message />
        </ErrorPrimitive.Root>
      )}

      <div className="mt-1 flex items-center gap-2 pl-1 text-muted">
        <ActionBarPrimitive.Root
          autohide="not-last"
          className="flex gap-2 opacity-0 transition group-hover:opacity-100"
        >
          <ActionBarPrimitive.Copy className="hover:text-ink" aria-label="Copy reply">
            <AuiIf condition={(s) => s.message.isCopied}>
              <CheckIcon width={13} height={13} />
            </AuiIf>
            <AuiIf condition={(s) => !s.message.isCopied}>
              <CopyIcon width={13} height={13} />
            </AuiIf>
          </ActionBarPrimitive.Copy>
          <ActionBarPrimitive.Reload className="hover:text-ink" aria-label="Regenerate reply">
            <RefreshIcon width={13} height={13} />
          </ActionBarPrimitive.Reload>
        </ActionBarPrimitive.Root>
        {timing?.totalStreamTime !== undefined && (
          <span className="text-[11px] tnum opacity-0 transition group-hover:opacity-100">
            {(timing.totalStreamTime / 1000).toFixed(1)}s
            {timing.tokensPerSecond ? ` · ${timing.tokensPerSecond.toFixed(0)} tok/s` : ''}
          </span>
        )}
      </div>
    </MessagePrimitive.Root>
  );
}

// Meal-summary card for the `logmeal` tool (see
// .context/prompts/chat-assistant-v2/08a-chat-logmeal-tool.md for the backend
// reply format this parses: "Logged: <raw>\n<kcal> kcal · <p>g protein ·
// <c>g carbs · <f>g fat"). Falls back to the generic ToolCallChip if the
// backend's wording doesn't match — the format isn't a public contract, this
// is a best-effort upgrade, not something that should ever break the message.
const LOGMEAL_RESULT_RE =
  /^Logged: (.+)\n([\d.]+) kcal(?:\D+([\d.]+)g protein)?(?:\D+([\d.]+)g carbs)?(?:\D+([\d.]+)g fat)?/

function LogMealToolCard(props: ToolCallMessagePartProps) {
  const { result } = props
  const parsed = typeof result === 'string' ? LOGMEAL_RESULT_RE.exec(result) : null
  const aui = useAui()

  if (!parsed) return <ToolCallChip {...props} />
  const [, rawText, kcal, protein, carbs, fat] = parsed

  return (
    <motion.div
      variants={fadeUp}
      initial="hidden"
      animate="show"
      className="my-1.5 rounded-lg border border-line bg-surface px-3 py-2.5"
    >
      <p className="text-sm font-medium text-ink">{rawText}</p>
      <div className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-xs tnum">
        <span className="text-[--color-cal]">{Math.round(Number(kcal))} kcal</span>
        {protein && <span className="text-[--color-protein]">{Math.round(Number(protein))}g protein</span>}
        {carbs && <span className="text-[--color-carbs]">{Math.round(Number(carbs))}g carbs</span>}
        {fat && <span className="text-[--color-fat]">{Math.round(Number(fat))}g fat</span>}
      </div>
      <button
        type="button"
        onClick={() => aui.thread().composer().setText(`Actually, log "${rawText}" as `)}
        className="mt-2 text-xs font-medium text-primary hover:underline"
      >
        Log a different amount
      </button>
    </motion.div>
  )
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
