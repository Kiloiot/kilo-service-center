import type { DownloadableFile } from "@api-types/files";
import { useMutation } from "@tanstack/react-query";

import { downloadBlob } from "@utils/downloadBlob";

/** A download the user triggers: fetches the file, then hands it to the browser. */
export function useFileDownload<TArgs, TFile extends DownloadableFile>(
  fetchFile: (args: TArgs) => Promise<TFile>,
) {
  return useMutation({
    mutationFn: fetchFile,
    onSuccess: (file) => downloadBlob(file.blob, file.filename),
  });
}
