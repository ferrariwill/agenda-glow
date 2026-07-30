import { useEffect, useMemo, useState } from 'react'

import { Link, useNavigate, useParams } from 'react-router-dom'

import { Clock, Sparkles } from 'lucide-react'

import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'

import { Alert } from '../../components/ui/Alert'

import { Button } from '../../components/ui/Button'

import { DateInput } from '../../components/ui/DateInput'

import { useAuth } from '../../contexts/AuthContext'

import { IS_MOCK } from '../../lib/config'

import { useStoreDb } from '../../data/store'

import { fetchPublicSlots } from '../../data/sync'

import { enviarConfirmacaoAgendamento } from '../../services/whatsappService'

import {

  AgendaConflitoError,

  createAgendamento,

  ensureClienteNoTenant,

  getHorariosDisponiveis,

  getServicosDoProfissional,

  getTenantBySlug,

} from '../../utils/mockDb'

import { formatBRL, formatDateTimeBR, formatTimeBR, initials, todayISO } from '../../utils/format'



const GLASS =

  'rounded-xl border border-[#e5d3c8]/30 bg-white/80 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'



export function ClienteAgendar() {

  const { slug } = useParams<{ slug: string }>()

  const navigate = useNavigate()

  const { session } = useAuth()

  const tenant = slug ? getTenantBySlug(slug) : undefined

  const db = useStoreDb()



  const [profId, setProfId] = useState('')

  const [servicoIds, setServicoIds] = useState<string[]>([])

  const [data, setData] = useState(todayISO())

  const [hora, setHora] = useState('')

  const [aceitaAdiantar, setAceitaAdiantar] = useState(false)

  const [loading, setLoading] = useState(false)

  const [error, setError] = useState('')

  const [done, setDone] = useState(false)



  const profissionais = tenant

    ? db.profissionais.filter((p) => p.tenant_id === tenant.id && p.ativo)

    : []

  const allServicos = tenant ? db.servicos.filter((s) => s.tenant_id === tenant.id) : []



  const servicosProf = useMemo(

    () => (profId && tenant ? getServicosDoProfissional(tenant.id, profId, allServicos) : []),

    [profId, tenant?.id, allServicos],

  )



  useEffect(() => {

    if (!profId && profissionais[0]) {

      setProfId(profissionais[0].id)

    }

  }, [profissionais, profId])



  useEffect(() => {

    if (servicosProf.length === 0) {

      setServicoIds([])

      return

    }

    setServicoIds((prev) => {

      const valid = prev.filter((id) => servicosProf.some((s) => s.id === id))

      if (valid.length > 0) return valid

      return [servicosProf[0].id]

    })

  }, [servicosProf])



  const servicosSelecionados = useMemo(

    () => servicosProf.filter((s) => servicoIds.includes(s.id)),

    [servicosProf, servicoIds],

  )

  const totalDuracao = servicosSelecionados.reduce((s, x) => s + x.duracao_minutos, 0)

  const totalPreco = servicosSelecionados.reduce((s, x) => s + x.preco, 0)



  const [horarios, setHorarios] = useState<string[]>([])

  useEffect(() => {
    if (!profId || servicoIds.length === 0 || totalDuracao === 0) {
      setHorarios([])
      return
    }
    if (IS_MOCK) {
      setHorarios(getHorariosDisponiveis(profId, data, totalDuracao))
      return
    }
    if (!slug) {
      setHorarios([])
      return
    }
    let cancelled = false
    fetchPublicSlots(slug, profId, data, servicoIds[0])
      .then((slots) => {
        if (!cancelled) setHorarios(slots)
      })
      .catch(() => {
        if (!cancelled) setHorarios([])
      })
    return () => {
      cancelled = true
    }
  }, [profId, data, totalDuracao, servicoIds, slug, db.agendamentos])

  const toggleServico = (id: string) => {

    setServicoIds((prev) => {

      if (prev.includes(id)) {

        const next = prev.filter((x) => x !== id)

        return next.length > 0 ? next : prev

      }

      return [...prev, id]

    })

    setHora('')

  }



  const selectProf = (id: string) => {

    setProfId(id)

    setHora('')

  }



  const confirmar = async () => {

    if (!tenant || !session?.user || !profId || !hora || servicoIds.length === 0) return

    setError('')

    setLoading(true)

    try {

      ensureClienteNoTenant(tenant.id, {

        nome: session.user.nome,

        telefone: session.user.telefone ?? '',

        email: session.user.email,

      })

      const ag = await createAgendamento({

        tenant_id: tenant.id,

        profissional_id: profId,

        servico_ids: servicoIds,

        cliente_nome: session.user.nome,

        cliente_telefone: session.user.telefone ?? '',

        data,

        hora_inicio: hora,

        aceita_adiantar: aceitaAdiantar,

      }, { publicSlug: slug })



      const prof = profissionais.find((p) => p.id === profId)

      await enviarConfirmacaoAgendamento({

        telefone: session.user.telefone ?? '',

        clienteNome: session.user.nome,

        servico: servicosSelecionados.map((s) => s.nome).join(' + '),

        profissional: prof?.nome ?? '',

        data,

        hora,

      })



      if (ag.status === 'EM_APROVACAO') {

        setError('Horário sujeito a aprovação — você receberá um link no WhatsApp.')

      }

      setDone(true)

    } catch (err) {

      if (err instanceof AgendaConflitoError) setError(err.message)

      else setError(err instanceof Error ? err.message : 'Erro ao agendar')

    } finally {

      setLoading(false)

    }

  }



  if (!tenant) {

    return <p className="p-8 text-center text-[#514440]">Salão não encontrado.</p>

  }



  if (done) {

    return (

      <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`}>

        <div className={[GLASS, 'p-6 text-center'].join(' ')}>

          <Sparkles className="mx-auto mb-3 h-8 w-8 text-[#7d5141]" />

          <h1 className="font-display text-2xl font-semibold text-[#7d5141]">Agendamento confirmado!</h1>

          <p className="mt-2 text-sm text-[#514440]">

            {formatDateTimeBR(data, hora)} · {servicosSelecionados.map((s) => s.nome).join(' + ')}

          </p>

          {aceitaAdiantar && (

            <p className="mt-2 text-sm text-[#514440]">

              Você será avisada por WhatsApp se surgir um horário mais cedo.

            </p>

          )}

          <p className="mt-1 text-sm text-[#514440]">Enviamos a confirmação por WhatsApp.</p>

          <div className="mt-5 flex flex-col gap-2">

            <Button onClick={() => navigate(`/${slug}/conta`)} className="bg-[#7d5141] hover:bg-[#996958]">

              Ver meus agendamentos

            </Button>

            <Button variant="secondary" onClick={() => navigate(`/${slug}`)}>

              Voltar ao salão

            </Button>

          </div>

        </div>

      </ClienteSalaoLayout>

    )

  }



  return (

    <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`}>

      <h1 className="mb-1 font-display text-2xl font-semibold text-[#1a1c1c]">Agendar horário</h1>

      <p className="mb-5 text-sm text-[#514440]">

        Escolha a profissional, os serviços e o melhor horário para você.

      </p>



      {error && (

        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>

          {error}

        </Alert>

      )}



      <div className="space-y-4">

        <section className={GLASS + ' p-4'}>

          <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">

            Profissional e serviços

          </h2>

          <div className="space-y-3">

            {profissionais.map((p) => {

              const esp = db.especialidades.find((e) => e.id === p.especialidade_id)

              const servicos = getServicosDoProfissional(tenant.id, p.id, allServicos)

              const selected = p.id === profId

              return (

                <div

                  key={p.id}

                  className={[

                    'overflow-hidden rounded-xl border transition-colors',

                    selected

                      ? 'border-[#7d5141] bg-[#efdcd1]/20'

                      : 'border-[#efdcd1]/40 bg-[#faf9f8]/50',

                  ].join(' ')}

                >

                  <button

                    type="button"

                    onClick={() => selectProf(p.id)}

                    className="flex w-full items-center gap-3 px-4 py-3 text-left"

                  >

                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-xs font-bold text-[#7d5141]">

                      {initials(p.nome)}

                    </div>

                    <div className="min-w-0 flex-1">

                      <p className="font-semibold text-[#1a1c1c]">{p.nome}</p>

                      <p className="text-xs text-[#514440]">{esp?.nome}</p>

                    </div>

                    <span

                      className={[

                        'h-4 w-4 shrink-0 rounded-full border-2',

                        selected ? 'border-[#7d5141] bg-[#7d5141]' : 'border-[#d6c2bd]',

                      ].join(' ')}

                    />

                  </button>



                  {selected && servicos.length > 0 && (

                    <div className="border-t border-[#efdcd1]/40 px-4 pb-4 pt-2">

                      <p className="mb-2 text-xs font-medium text-[#514440]">Serviços disponíveis</p>

                      <div className="space-y-1">

                        {servicos.map((s) => {

                          const checked = servicoIds.includes(s.id)

                          return (

                            <label

                              key={s.id}

                              className={[

                                'flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2 text-sm',

                                checked ? 'bg-[#efdcd1]/40' : 'hover:bg-white/60',

                              ].join(' ')}

                            >

                              <input

                                type="checkbox"

                                checked={checked}

                                onChange={() => toggleServico(s.id)}

                                className="rounded border-[#d6c2bd] text-[#7d5141]"

                              />

                              <span className="flex-1 font-medium text-[#1a1c1c]">{s.nome}</span>

                              <span className="text-xs text-[#514440]">

                                {s.duracao_minutos} min · {formatBRL(s.preco)}

                              </span>

                            </label>

                          )

                        })}

                      </div>

                    </div>

                  )}



                  {selected && servicos.length === 0 && (

                    <p className="border-t border-[#efdcd1]/40 px-4 py-3 text-sm text-[#514440]">

                      Nenhum serviço online para esta profissional.

                    </p>

                  )}

                </div>

              )

            })}

          </div>



          {servicosSelecionados.length > 0 && (

            <p className="mt-3 text-sm font-medium text-[#7d5141]">

              Total: {totalDuracao} min · {formatBRL(totalPreco)}

            </p>

          )}

        </section>



        <section className={GLASS + ' p-4'}>

          <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">Data</h2>

          <DateInput

            label=""

            value={data}

            onChange={(d) => {

              setData(d)

              setHora('')

            }}

            minDate={todayISO()}

          />

        </section>



        {profId && servicoIds.length > 0 && (

          <section className={GLASS + ' p-4'}>

            <div className="mb-3 flex items-center gap-2">

              <Clock className="h-4 w-4 text-[#7d5141]" />

              <p className="text-sm font-bold uppercase tracking-widest text-[#514440]">

                Horários disponíveis

              </p>

            </div>

            {horarios.length === 0 ? (

              <p className="text-sm text-[#514440]">Nenhum horário livre nesta data.</p>

            ) : (

              <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">

                {horarios.map((slot) => (

                  <button

                    key={slot}

                    type="button"

                    onClick={() => setHora(slot)}

                    className={[

                      'rounded-lg border py-2.5 text-sm font-medium transition-colors',

                      hora === slot

                        ? 'border-[#7d5141] bg-[#7d5141] text-white'

                        : 'border-[#d6c2bd]/50 bg-white text-[#1a1c1c] hover:border-[#7d5141]',

                    ].join(' ')}

                  >

                    {formatTimeBR(slot)}

                  </button>

                ))}

              </div>

            )}

          </section>

        )}



        {hora && (

          <section className={GLASS + ' p-4'}>

            <label className="flex cursor-pointer items-start gap-3">

              <input

                type="checkbox"

                checked={aceitaAdiantar}

                onChange={(e) => setAceitaAdiantar(e.target.checked)}

                className="mt-1 rounded border-[#d6c2bd] text-[#7d5141]"

              />

              <span>

                <span className="block text-sm font-medium text-[#1a1c1c]">

                  Aceito adiantar meu horário se possível

                </span>

                <span className="mt-0.5 block text-xs text-[#514440]">

                  Se surgir uma vaga mais cedo com {profissionais.find((p) => p.id === profId)?.nome}, avisamos

                  pelo WhatsApp.

                </span>

              </span>

            </label>

          </section>

        )}



        {hora && (

          <Alert variant="info">

            Resumo: <strong>{formatDateTimeBR(data, hora)}</strong>

            {aceitaAdiantar ? ' · com opção de adiantar' : ''}

          </Alert>

        )}



        <Button

          fullWidth

          loading={loading}

          disabled={!hora || servicoIds.length === 0}

          onClick={confirmar}

          className="bg-[#7d5141] hover:bg-[#996958]"

        >

          Confirmar agendamento

        </Button>



        <p className="text-center text-xs text-[#514440]">

          {session?.user.nome} · {session?.user.telefone}{' '}

          <Link to={`/${slug}/conta`} className="text-[#7d5141] hover:underline">

            Minha conta

          </Link>

        </p>

      </div>

    </ClienteSalaoLayout>

  )

}


