export function bookingWhatsAppCopy(
  notificationsAvailable: boolean | undefined,
  optedIn: boolean,
): { earlySlot?: string; confirmation: string } {
  if (notificationsAvailable === false) {
    return {
      earlySlot: optedIn
        ? 'Sua preferência foi salva. Os avisos de horários mais cedo começarão quando o salão reativar o WhatsApp.'
        : undefined,
      confirmation:
        'Agendamento salvo. As mensagens pelo WhatsApp voltarão quando o salão reativar o canal.',
    }
  }

  if (notificationsAvailable === true) {
    return {
      earlySlot: optedIn
        ? 'Você será avisada por WhatsApp se surgir um horário mais cedo.'
        : undefined,
      confirmation: 'Enviamos a confirmação por WhatsApp.',
    }
  }

  return {
    earlySlot: optedIn
      ? 'Sua preferência por um horário mais cedo foi salva.'
      : undefined,
    confirmation: 'Agendamento salvo com sucesso.',
  }
}
