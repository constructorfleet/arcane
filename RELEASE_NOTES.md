# 3.0.1-cf.47

- Preserve authored Compose, env and project override text through stack/project/template source form validation, including final newlines and leading/trailing blank lines (#113).
- Keep normal field trimming and reject whitespace-only required Compose without rewriting valid source. Project creation retains whitespace-only env text in its multipart request, file writer and direct env edits; imported templates also persist a new whitespace-only env file.

Local verification checks exact form/request values, native project/template file bytes and existing frontend coverage; runtime source-save acceptance remains separate. No deployed application stack or global default template is edited by local validation. Broader Compose/template acceptance remains #28/#17.

Git-managed effective env/override derivation keeps its existing semantic merge behavior; this change does not promise raw byte preservation of derived files.
