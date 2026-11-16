## Workload Profiles

```yaml
- uri: FIXME
  type: FIXME
  class: FIXME
  format: FIXME
  description: FIXME
  outputs:
    FIXME: FIXME
  manifests: |
    FIXME
```

Access to:
- `SourceWorkload`: `string`
- `Location`: `string`
- `ProjectId`: `string`

Template functions:
- `hclescape`: escapes a value for use inside an HCL double-quoted string (`"`, `\`, newlines and control characters). Use it for any user-supplied value in Terraform manifests, e.g. `value = "{{ hclescape $variableValue }}"`. `${...}` is left as-is so Terraform interpolation still works.
