# 3.0.1-cf.55

- Preserve untouched standalone memory and memory-plus-swap limits by omitting them from unrelated edits (#129). Display fractional MiB values without rounding away native bytes.
- Support Docker's -1 unlimited-swap sentinel in create/edit forms, with guidance and validation for the supported negative value. Explicit zero still resets the swap setting.

Validation covers the actual owning form, untouched native values and intentional resource edits. Live acceptance uses only an owned isolated container; no production workload is edited. Broader #28 remains open.
