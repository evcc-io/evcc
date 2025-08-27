// Copy text to clipboard, returns true if successful
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error("Failed to copy to clipboard:", err);
    return false;
  }
}

// Copy text to clipboard and manage a "copied" state with timeout
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
