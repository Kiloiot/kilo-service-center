import type { DownloadableFile } from "@api-types/files";

import { TIMING_DOWNLOAD_URL_REVOKE_MS } from "@constants/app";

/** Wraps downloaded bytes as a file the browser can save. */
export function fileFromBytes(
  content: Uint8Array,
  filename: string,
  contentType: string,
): DownloadableFile {
  return {
    blob: new Blob([new Uint8Array(content)], { type: contentType }),
    filename,
  };
}

/** Trigger a browser download of a blob under the given file name. */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  setTimeout(() => URL.revokeObjectURL(url), TIMING_DOWNLOAD_URL_REVOKE_MS);
}
