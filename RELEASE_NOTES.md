# 3.0.1-cf.54

- Author standalone command and entrypoint arrays as literal argument rows, matching Swarm services (#127). Preserve order, quotes, spaces, empty values, duplicates and LF text without shell-word parsing.
- Distinguish no rows/image defaults from an intentional empty argument. Keep the original-snapshot omission of untouched execution settings and native healthcheck preservation from cf.53.

Validation covers actual owning create/edit forms and native payload contracts. Live acceptance will use only an owned isolated container; no production workload is edited. Healthcheck authoring is unchanged; browser textarea normalization is outside an arbitrary control-character byte-preservation claim. Broader #28 remains open.
