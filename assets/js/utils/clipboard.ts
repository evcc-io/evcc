export function isClipboardSupported(): boolean {
  return !!(navigator.clipboard && window.isSecureContext);
}

export async function copyToClipboard(text: string): Promise<boolean> {
  if (!isClipboardSupported()) {
    return false;
  }
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error("Failed to copy to clipboard:", err);
    return false;
  }
}

export async function copyWithFeedback(
  text: string,
  setCopiedState: (value: boolean) => void
): Promise<void> {
  const success = await copyToClipboard(text);
  if (success) {
    setCopiedState(true);
    setTimeout(() => {
      setCopiedState(false);
    }, 2000);
  }
}
