# 3.0.1-cf.61

- Render repeated template-registry tag badges without duplicate-key errors in the gallery (#142 / #17). Reuse the existing occurrence keys while preserving the first four tags' order, repeated values and overflow count.

Verified with the actual owning client component under development and production runtime conditions using an isolated DOM. No live registry/template was changed; broader template deployment and branding acceptance remains separate.
