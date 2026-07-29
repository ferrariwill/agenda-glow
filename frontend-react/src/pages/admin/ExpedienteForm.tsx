import type { Expediente } from '../../types'
import { TimeInput } from '../../components/ui/TimeInput'

const DIAS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

interface ExpedienteFormProps {
  value: Expediente[]
  onChange: (expedientes: Expediente[]) => void
}

const emptyDay = (dia: number): Expediente => ({
  dia_semana: dia,
  horario_entrada: '09:00',
  horario_saida: '18:00',
  inicio_almoco: '12:00',
  fim_almoco: '13:00',
})

export function ExpedienteForm({ value, onChange }: ExpedienteFormProps) {
  const getDay = (dia: number) => value.find((e) => e.dia_semana === dia)

  const toggleDay = (dia: number, active: boolean) => {
    if (active) {
      onChange([...value.filter((e) => e.dia_semana !== dia), getDay(dia) ?? emptyDay(dia)])
    } else {
      onChange(value.filter((e) => e.dia_semana !== dia))
    }
  }

  const updateDay = (dia: number, patch: Partial<Expediente>) => {
    onChange(
      value.map((e) => (e.dia_semana === dia ? { ...e, ...patch, dia_semana: dia } : e)),
    )
  }

  const validateJornada = (exp: Expediente): string | null => {
    const entrada = exp.horario_entrada
    const saida = exp.horario_saida
    if (saida <= entrada) return 'Saída deve ser após a entrada'
    if (exp.inicio_almoco && exp.fim_almoco && exp.fim_almoco <= exp.inicio_almoco) {
      return 'Fim do almoço deve ser após o início'
    }
    return null
  }

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium text-aura-anthracite">Expediente semanal</p>
      {DIAS.map((label, dia) => {
        const exp = getDay(dia)
        const active = !!exp
        const err = exp ? validateJornada(exp) : null
        return (
          <div
            key={dia}
            className="rounded-lg border border-aura-border p-3"
          >
            <label className="mb-2 flex items-center gap-2 text-sm font-medium">
              <input
                type="checkbox"
                checked={active}
                onChange={(e) => toggleDay(dia, e.target.checked)}
                className="rounded border-aura-border"
              />
              {label}
            </label>
            {active && exp && (
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                <TimeInput
                  label="Entrada"
                  value={exp.horario_entrada}
                  onChange={(v) => updateDay(dia, { horario_entrada: v })}
                  className="text-sm"
                />
                <TimeInput
                  label="Saída"
                  value={exp.horario_saida}
                  onChange={(v) => updateDay(dia, { horario_saida: v })}
                  className="text-sm"
                />
                <TimeInput
                  label="Almoço início"
                  value={exp.inicio_almoco ?? ''}
                  onChange={(v) => updateDay(dia, { inicio_almoco: v })}
                  className="text-sm"
                />
                <TimeInput
                  label="Almoço fim"
                  value={exp.fim_almoco ?? ''}
                  onChange={(v) => updateDay(dia, { fim_almoco: v })}
                  className="text-sm"
                />
              </div>
            )}
            {err && <p className="mt-1 text-xs text-red-600">{err}</p>}
          </div>
        )
      })}
    </div>
  )
}

export function validateAllExpedientes(expedientes: Expediente[]): string | null {
  for (const exp of expedientes) {
    if (exp.horario_saida <= exp.horario_entrada) {
      return `Dia ${DIAS[exp.dia_semana]}: saída deve ser após entrada`
    }
  }
  return null
}
