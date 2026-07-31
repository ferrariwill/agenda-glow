import type { ReactNode } from 'react'
import { Inbox } from 'lucide-react'

export type EntityColumnPriority = 'primary' | 'secondary'

export type EntityColumn<T> = {
  header: string
  cell: (item: T) => ReactNode
  align?: 'left' | 'right'
  /** secondary columns hide until lg when tableFrom is md (dense tables). */
  priority?: EntityColumnPriority
}

export type ResponsiveEntityListProps<T> = {
  items: T[]
  getKey: (item: T) => string
  /** Mobile (&lt; tableFrom): card com id + ≤2 decisões + ação. */
  renderCard: (item: T, index: number) => ReactNode
  columns: EntityColumn<T>[]
  emptyTitle: string
  emptyDescription?: string
  emptyAction?: ReactNode
  loading?: boolean
  /** Breakpoint from which the table is shown. Default md (768). */
  tableFrom?: 'md' | 'lg'
  className?: string
  /** Optional wrapper class for the card list region. */
  cardListClassName?: string
  /** Optional wrapper class for the table region. */
  tableClassName?: string
}

const tableVisible: Record<'md' | 'lg', string> = {
  md: 'hidden md:block',
  lg: 'hidden lg:block',
}

const cardsVisible: Record<'md' | 'lg', string> = {
  md: 'md:hidden',
  lg: 'lg:hidden',
}

const secondaryColVisible = 'hidden lg:table-cell'

function SkeletonCards({ count = 4 }: { count?: number }) {
  return (
    <div className="space-y-3 p-3" aria-hidden>
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="animate-pulse rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4"
        >
          <div className="flex items-start gap-3">
            <div className="h-10 w-10 shrink-0 rounded-full bg-[#efdcd1]/60" />
            <div className="min-w-0 flex-1 space-y-2">
              <div className="h-4 w-3/4 rounded bg-[#efdcd1]/60" />
              <div className="h-3 w-1/2 rounded bg-[#e9e8e7]" />
            </div>
          </div>
          <div className="mt-3 flex justify-between gap-4">
            <div className="h-3 w-24 rounded bg-[#e9e8e7]" />
            <div className="h-4 w-16 rounded bg-[#efdcd1]/60" />
          </div>
        </div>
      ))}
    </div>
  )
}

function SkeletonRows<T>({
  columns,
  count = 5,
}: {
  columns: EntityColumn<T>[]
  count?: number
}) {
  return (
    <tbody aria-hidden>
      {Array.from({ length: count }, (_, i) => (
        <tr key={i} className="animate-pulse">
          {columns.map((col) => (
            <td
              key={col.header}
              className={[
                'px-5 py-4',
                col.priority === 'secondary' ? secondaryColVisible : '',
              ]
                .filter(Boolean)
                .join(' ')}
            >
              <div className="h-4 w-full max-w-[8rem] rounded bg-[#efdcd1]/50" />
            </td>
          ))}
        </tr>
      ))}
    </tbody>
  )
}

function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-5 py-12 text-center">
      <div className="rounded-full bg-[#efdcd1]/40 p-3 text-[#7d5141]">
        <Inbox className="h-6 w-6" aria-hidden />
      </div>
      <p className="text-sm font-semibold text-[#7d5141]">{title}</p>
      {description && <p className="max-w-sm text-sm text-aura-muted">{description}</p>}
      {action}
    </div>
  )
}

/**
 * Listagem canônica: cards &lt; breakpoint, tabela ≥ breakpoint.
 * Extrai o padrão dual de ClientesDona (md:hidden / hidden md:block).
 */
export function ResponsiveEntityList<T>({
  items,
  getKey,
  renderCard,
  columns,
  emptyTitle,
  emptyDescription,
  emptyAction,
  loading = false,
  tableFrom = 'md',
  className = '',
  cardListClassName = 'space-y-3 p-3',
  tableClassName = '',
}: ResponsiveEntityListProps<T>) {
  const isEmpty = !loading && items.length === 0

  return (
    <div className={className}>
      {/* Mobile / narrow: cards only */}
      <div className={cardsVisible[tableFrom]}>
        {loading ? (
          <SkeletonCards />
        ) : isEmpty ? (
          <EmptyState title={emptyTitle} description={emptyDescription} action={emptyAction} />
        ) : (
          <div className={cardListClassName}>
            {items.map((item, index) => (
              <div key={getKey(item)}>{renderCard(item, index)}</div>
            ))}
          </div>
        )}
      </div>

      {/* Desktop / wide: table — no overflow-x wrapper; dense cols use priority */}
      <div className={[tableVisible[tableFrom], tableClassName].filter(Boolean).join(' ')}>
        {isEmpty && !loading ? (
          <EmptyState title={emptyTitle} description={emptyDescription} action={emptyAction} />
        ) : (
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="bg-[#f4f3f2]">
                {columns.map((col) => (
                  <th
                    key={col.header}
                    className={[
                      'border-b border-[#e9e8e7] px-5 py-3 text-[10px] font-bold uppercase tracking-wider text-aura-muted',
                      col.align === 'right' ? 'text-right' : '',
                      col.priority === 'secondary' ? secondaryColVisible : '',
                    ]
                      .filter(Boolean)
                      .join(' ')}
                  >
                    {col.header}
                  </th>
                ))}
              </tr>
            </thead>
            {loading ? (
              <SkeletonRows columns={columns} />
            ) : (
              <tbody className="divide-y divide-[#e9e8e7]">
                {items.map((item) => (
                  <tr
                    key={getKey(item)}
                    className="group transition-colors hover:bg-[#faf9f8]"
                  >
                    {columns.map((col) => (
                      <td
                        key={col.header}
                        className={[
                          'px-5 py-4',
                          col.align === 'right' ? 'text-right' : '',
                          col.priority === 'secondary' ? secondaryColVisible : '',
                        ]
                          .filter(Boolean)
                          .join(' ')}
                      >
                        {col.cell(item)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            )}
          </table>
        )}
      </div>
    </div>
  )
}
