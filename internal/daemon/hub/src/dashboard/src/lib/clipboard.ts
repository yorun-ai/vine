export async function copyTextToClipboard(value: string) {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // Clipboard API can be unavailable or denied outside a secure context.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.readOnly = true
  textarea.style.position = 'fixed'
  textarea.style.inset = '0 auto auto -9999px'
  textarea.style.opacity = '0'
  document.body.append(textarea)

  try {
    textarea.select()
    textarea.setSelectionRange(0, value.length)
    if (!document.execCommand('copy')) {
      throw new Error('Copy failed')
    }
  } finally {
    textarea.remove()
  }
}
