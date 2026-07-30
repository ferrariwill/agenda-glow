export function visibleOfferError(loadError: string, actionError: string): string {
  return actionError || loadError
}

export function softReloadResult(
  actionError: string,
  nextLoadError: string,
): { actionError: string; loadError: string } {
  return { actionError, loadError: nextLoadError }
}
