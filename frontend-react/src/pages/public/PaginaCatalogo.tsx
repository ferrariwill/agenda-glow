import { Link, useNavigate, useParams } from 'react-router-dom'

import { Calendar, Sparkles } from 'lucide-react'

import { Button } from '../../components/ui/Button'

import { useAuth } from '../../contexts/AuthContext'

import { getServicosDoProfissional, getTenantBySlug } from '../../utils/mockDb'
import { useStoreDb } from '../../data/store'

import { formatBRL, initials } from '../../utils/format'



const GLASS =

  'rounded-xl border border-[#e5d3c8]/30 bg-white/80 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'



export function PaginaCatalogo() {

  const { slug } = useParams<{ slug: string }>()

  const navigate = useNavigate()

  const { session } = useAuth()

  const tenant = slug ? getTenantBySlug(slug) : undefined

  const db = useStoreDb()



  if (!tenant || tenant.status !== 'ATIVO') {

    return (

      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8] p-4">

        <p className="text-[#514440]">Salão não encontrado ou indisponível.</p>

      </div>

    )

  }



  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenant.id && p.ativo)

  const isCliente = session?.user.role === 'CLIENTE'



  const handleAgendar = () => {

    if (isCliente) {

      navigate(`/${slug}/agendar`)

    } else {

      navigate(`/${slug}/cadastro`)

    }

  }



  return (

    <div className="min-h-screen bg-[#faf9f8]">

      <header className="relative overflow-hidden bg-gradient-to-b from-[#efdcd1]/60 to-[#faf9f8] px-4 pb-10 pt-10 text-center">

        {tenant.logo_url ? (

          <img

            src={tenant.logo_url}

            alt={tenant.nome}

            className="mx-auto mb-4 h-24 w-24 rounded-full border-4 border-white object-cover shadow-lg"

          />

        ) : (

          <div className="mx-auto mb-4 flex h-24 w-24 items-center justify-center rounded-full bg-[#7d5141] text-2xl font-bold text-white shadow-lg">

            {tenant.nome.slice(0, 2).toUpperCase()}

          </div>

        )}

        <h1 className="font-display text-2xl font-semibold text-[#1a1c1c] sm:text-3xl">{tenant.nome}</h1>

        <p className="mx-auto mt-2 max-w-md text-sm text-[#514440]">{tenant.bio}</p>

        <Sparkles className="mx-auto mt-4 h-5 w-5 text-[#7d5141]" />

      </header>



      <main className="mx-auto max-w-lg space-y-5 px-4 pb-[max(3rem,env(safe-area-inset-bottom))] -mt-4">

        <section className={GLASS + ' p-5'}>

          <h2 className="mb-4 font-display text-xl font-semibold text-[#7d5141]">Nossa equipe</h2>

          <div className="space-y-4">

            {profissionais.map((p) => {

              const esp = db.especialidades.find((e) => e.id === p.especialidade_id)

              const servicosProf = getServicosDoProfissional(tenant.id, p.id, db.servicos)

              return (

                <div

                  key={p.id}

                  className="rounded-lg border border-[#efdcd1]/40 bg-[#faf9f8]/80 p-4"

                >

                  <div className="flex items-center gap-3">

                    <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-sm font-bold text-[#7d5141]">

                      {initials(p.nome)}

                    </div>

                    <div>

                      <p className="font-semibold text-[#1a1c1c]">{p.nome}</p>

                      <p className="text-xs font-medium uppercase tracking-wider text-[#514440]">

                        {esp?.nome ?? 'Profissional'}

                      </p>

                    </div>

                  </div>

                  {servicosProf.length > 0 && (

                    <ul className="mt-3 space-y-2 border-t border-[#efdcd1]/30 pt-3">

                      {servicosProf.slice(0, 4).map((s) => (

                        <li key={s.id} className="flex items-center justify-between text-sm">

                          <span className="text-[#514440]">{s.nome}</span>

                          <span className="font-medium text-[#7d5141]">

                            {s.duracao_minutos} min · {formatBRL(s.preco)}

                          </span>

                        </li>

                      ))}

                      {servicosProf.length > 4 && (

                        <li className="text-xs text-[#514440]">

                          +{servicosProf.length - 4} serviços

                        </li>

                      )}

                    </ul>

                  )}

                </div>

              )

            })}

          </div>

        </section>



        <Button

          fullWidth

          onClick={handleAgendar}

          className="bg-[#7d5141] py-3.5 shadow-lg shadow-[#7d5141]/20 hover:bg-[#996958]"

        >

          <Calendar className="h-4 w-4" />

          Agendar horário

        </Button>



        <div className="flex justify-center gap-4 text-sm">

          {isCliente ? (

            <Link to={`/${slug}/conta`} className="text-[#7d5141] hover:underline">

              Minha conta

            </Link>

          ) : (

            <>

              <Link to={`/${slug}/cadastro`} className="font-semibold text-[#7d5141] hover:underline">

                Primeiro agendamento

              </Link>

              <Link to={`/${slug}/login`} className="text-[#514440] hover:text-[#7d5141]">

                Já tenho conta

              </Link>

            </>

          )}

        </div>

      </main>

    </div>

  )

}


