import { writable, get } from 'svelte/store';

/**
 * Clipboard Service for ctrlvweb
 * Provides helper functions to write text and images to the system clipboard
 * across all modern web browsers.
 *
 * Browsers enforce Transient User Activation for navigator.clipboard.write().
 * This service handles direct writes, catches activation blocks gracefully,
 * stages pending images, and automatically triggers the copy on the user's
 * next interaction (click / tap / keydown).
 */

export const pendingClipboardImage = writable<string | null>(null);

let cleanupActivationListeners: (() => void) | null = null;

/**
 * Converts a base64 data URL or image source into an image/png Blob.
 * Standard browser ClipboardItem API strictly requires the MIME type 'image/png'
 * for writing images to clipboard.
 */
export async function convertToPngBlob(imageSrc: string): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      try {
        const canvas = document.createElement('canvas');
        canvas.width = img.naturalWidth || img.width;
        canvas.height = img.naturalHeight || img.height;
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          reject(new Error('Canvas 2D rendering context is unavailable'));
          return;
        }
        ctx.drawImage(img, 0, 0);
        canvas.toBlob((blob) => {
          if (blob) {
            resolve(blob);
          } else {
            reject(new Error('Canvas toBlob returned null'));
          }
        }, 'image/png');
      } catch (err) {
        reject(err);
      }
    };
    img.onerror = (e) => reject(new Error('Failed to load image for clipboard conversion: ' + e));
    img.src = imageSrc;
  });
}

export interface ClipboardWriteResult {
  success: boolean;
  error?: string;
  blockedByActivation?: boolean;
}

/**
 * Detailed image clipboard writer.
 * Passes blob promise to ClipboardItem to preserve user activation synchronously
 * where supported by the browser engine.
 */
export async function copyImageToClipboardDetailed(imageSrc: string): Promise<ClipboardWriteResult> {
  if (!imageSrc || typeof window === 'undefined') {
    return { success: false, error: 'No image data' };
  }
  if (!navigator.clipboard || !window.ClipboardItem) {
    console.warn('ClipboardItem API is not supported in this browser.');
    return { success: false, error: 'ClipboardItem API not supported' };
  }

  try {
    const blobPromise = convertToPngBlob(imageSrc);

    try {
      // Modern Chromium & WebKit allow passing Promise<Blob> directly to ClipboardItem
      await navigator.clipboard.write([
        new ClipboardItem({
          'image/png': blobPromise
        })
      ]);
      return { success: true };
    } catch (writeErr: any) {
      const msg = String(writeErr?.message || writeErr);
      const isActivation =
        msg.includes('user activation') ||
        msg.includes('must be focused') ||
        writeErr?.name === 'NotAllowedError';

      if (isActivation) {
        return { success: false, error: msg, blockedByActivation: true };
      }

      // Fallback: Await blob first and write directly
      const pngBlob = await blobPromise;
      await navigator.clipboard.write([
        new ClipboardItem({
          'image/png': pngBlob
        })
      ]);
      return { success: true };
    }
  } catch (err: any) {
    const msg = String(err?.message || err);
    const isActivation =
      msg.includes('user activation') ||
      msg.includes('must be focused') ||
      err?.name === 'NotAllowedError';

    if (!isActivation) {
      console.warn('Could not copy image to clipboard:', err);
    }
    return { success: false, error: msg, blockedByActivation: isActivation };
  }
}

/**
 * Standard boolean helper for user-initiated copy actions.
 */
export async function copyImageToClipboard(imageSrc: string): Promise<boolean> {
  const res = await copyImageToClipboardDetailed(imageSrc);
  return res.success;
}

/**
 * Triggers auto-copy for an incoming screenshot.
 * 1. Tries direct clipboard write (succeeds if tab has focus / active activation).
 * 2. If blocked by lack of user activation, queues the image in pendingClipboardImage
 *    and registers a one-time global interaction listener (pointerdown/keydown)
 *    so the very instant the user taps or clicks anywhere, the image is copied!
 */
export async function triggerAutoCopy(imageSrc: string): Promise<boolean> {
  if (!imageSrc || typeof window === 'undefined') return false;

  // 1. Direct attempt
  const directRes = await copyImageToClipboardDetailed(imageSrc);
  if (directRes.success) {
    clearPendingImage();
    window.dispatchEvent(new CustomEvent('ctrlv_image_auto_copied'));
    return true;
  }

  // 2. If blocked by lack of user activation, stage for immediate one-click copy
  pendingClipboardImage.set(imageSrc);
  window.dispatchEvent(new CustomEvent('ctrlv_clipboard_activation_needed'));

  // Clean up previous listeners if any
  if (cleanupActivationListeners) {
    cleanupActivationListeners();
    cleanupActivationListeners = null;
  }

  // 3. One-time interaction handler on window
  const onUserInteract = async () => {
    if (cleanupActivationListeners) {
      cleanupActivationListeners();
      cleanupActivationListeners = null;
    }

    const pending = get(pendingClipboardImage);
    if (!pending) return;

    const retryRes = await copyImageToClipboardDetailed(pending);
    if (retryRes.success) {
      clearPendingImage();
      window.dispatchEvent(new CustomEvent('ctrlv_image_auto_copied'));
    }
  };

  window.addEventListener('pointerdown', onUserInteract, { capture: true, once: true });
  window.addEventListener('keydown', onUserInteract, { capture: true, once: true });

  cleanupActivationListeners = () => {
    window.removeEventListener('pointerdown', onUserInteract, { capture: true });
    window.removeEventListener('keydown', onUserInteract, { capture: true });
  };

  return false;
}

/**
 * Copies the currently pending auto-copy image (invoked by user click/tap on banner).
 */
export async function copyPendingImage(): Promise<boolean> {
  const pending = get(pendingClipboardImage);
  if (!pending) return false;
  const res = await copyImageToClipboardDetailed(pending);
  if (res.success) {
    clearPendingImage();
    window.dispatchEvent(new CustomEvent('ctrlv_image_auto_copied'));
    return true;
  }
  return false;
}

/**
 * Clears the pending image and cleans up listeners.
 */
export function clearPendingImage() {
  pendingClipboardImage.set(null);
  if (cleanupActivationListeners) {
    cleanupActivationListeners();
    cleanupActivationListeners = null;
  }
}

export function dismissPendingImage() {
  clearPendingImage();
}

