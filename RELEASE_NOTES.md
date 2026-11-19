# 3.0.1-cf.11

## Features

- Preview PNG, JPEG and GIF files in the volume workspace as read-only images (#15).
- Bound previews to 5 MiB, 4096 pixels per side, and the configured workspace read limit. Check image headers and do not preview HTML, SVG, symlinks, special files, unsupported formats, or oversized images.
- Keep binary bytes separate from editable text. Show metadata and an explicit download action for unsupported files; decoding failures have a visible fallback.

## Validation

- Business tests cover supported formats, malformed headers, dimensions, file-size limits, and unchanged text/read-only semantics.
- Required CI and live preview/download acceptance precede completion. Volume attachment and RBD handoff policies are unchanged.
