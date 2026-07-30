import { earlySlotErrorMessage } from '../../services/earlySlotOfferApi'
import type { EarlySlotPreferenceResult } from '../../types/earlySlot'

export function resolvePreferenceSuccess(
  currentChannelAvailable: boolean | undefined,
  result: EarlySlotPreferenceResult,
) {
  const hasChannelEcho = typeof result.early_slot_notifications_available === 'boolean'

  return {
    checked: result.aceita_adiantar,
    channelAvailable: hasChannelEcho
      ? result.early_slot_notifications_available
      : currentChannelAvailable,
    hasChannelEcho,
    onChange: {
      aceitaAdiantar: result.aceita_adiantar,
      aceitaAdiantarEm: result.aceita_adiantar_em,
      notificationsAvailable: result.early_slot_notifications_available,
    },
  }
}

export function resolvePreferenceFailure(previousChecked: boolean, err: unknown) {
  return {
    checked: previousChecked,
    error: earlySlotErrorMessage(err),
  }
}
