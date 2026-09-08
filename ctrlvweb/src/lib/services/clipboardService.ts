/**
 * Clipboard Service for ctrlvweb
 * Provides helper functions to write text and images to the system clipboard
 * across all modern web browsers.
 */

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

/**
 * Copies an image to the system clipboard.
 * Converts to PNG Blob and uses navigator.clipboard.write([new ClipboardItem(...)])
 * Handles permission / focus failures gracefully without crashing.
 */
export async function copyImageToClipboard(imageSrc: string): Promise<boolean> {
  if (!imageSrc || typeof window === 'undefined') return false;
  try {
    if (!navigator.clipboard || !window.ClipboardItem) {
      console.warn('ClipboardItem API is not supported in this browser.');
      return false;
    }
    const pngBlob = await convertToPngBlob(imageSrc);
    await navigator.clipboard.write([
      new ClipboardItem({
        'image/png': pngBlob
      })
    ]);
    return true;
  } catch (err) {
    console.warn('Could not copy image to clipboard (document may not be focused or permission denied):', err);
    return false;
  }
}
