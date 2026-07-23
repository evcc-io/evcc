export function isClipboardSupported(): boolean {
  return !!(navigator.clipboard && window.isSecureContext);
}

// textarea/execCommand fallback for HTTP/non-secure contexts
function fallbackCopy(text: string): boolean {
  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.left = "-999999px";
    textarea.style.top = "-999999px";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    const successful = document.execCommand("copy");
    document.body.removeChild(textarea);
    return successful;
  } catch (err) {
    console.error("Fallback copy failed:", err);
    return false;
  }
}

export async function copyToClipboard(text: string): Promise<boolean> {
  if (!isClipboardSupported()) {
    return fallbackCopy(text);
  }
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error("Failed to copy to clipboard:", err);
    return fallbackCopy(text);
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
