const STEPS = [
  { id: 1, label: 'Profissional' },
  { id: 2, label: 'Serviços' },
  { id: 3, label: 'Data e hora' },
  { id: 4, label: 'Confirmar' },
] as const

export type BookingStepperProps = {
  current: 1 | 2 | 3 | 4
  onGoTo?: (step: 1 | 2 | 3 | 4) => void
  maxReachable?: 1 | 2 | 3 | 4
}

export function BookingStepper({ current, onGoTo, maxReachable = current }: BookingStepperProps) {
  return (
    <nav aria-label="Progresso do agendamento" className="mb-5">
      <ol className="flex items-center gap-1">
        {STEPS.map((step, index) => {
          const done = step.id < current
          const active = step.id === current
          const reachable = step.id <= maxReachable
          return (
            <li key={step.id} className="flex min-w-0 flex-1 items-center gap-1">
              <button
                type="button"
                disabled={!reachable || !onGoTo}
                onClick={() => reachable && onGoTo?.(step.id)}
                className={[
                  'flex min-h-touch-min min-w-0 flex-1 touch-manipulation flex-col items-center justify-center gap-0.5 rounded-lg px-0.5 text-center transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#7d5141]/40',
                  active
                    ? 'text-[#7d5141]'
                    : done
                      ? 'text-[#7d5141]/80'
                      : 'text-[#514440]/50',
                  reachable && onGoTo ? 'cursor-pointer' : 'cursor-default',
                ].join(' ')}
                aria-current={active ? 'step' : undefined}
              >
                <span
                  className={[
                    'flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold',
                    active || done
                      ? 'bg-[#7d5141] text-white'
                      : 'bg-[#efdcd1]/60 text-[#514440]',
                  ].join(' ')}
                  aria-hidden
                >
                  {done ? '✓' : step.id}
                </span>
                <span className="truncate text-xs font-medium leading-tight">
                  {step.label}
                </span>
              </button>
              {index < STEPS.length - 1 && (
                <span
                  className={[
                    'mb-4 h-px w-2 shrink-0 sm:w-3',
                    step.id < current ? 'bg-[#7d5141]' : 'bg-[#efdcd1]',
                  ].join(' ')}
                  aria-hidden
                />
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
