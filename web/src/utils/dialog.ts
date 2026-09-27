export function isDialogCancel(error: unknown) {
  return error === 'cancel' || error === 'close'
}
