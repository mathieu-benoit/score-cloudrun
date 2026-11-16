## Workload Profiles

```yaml
- id: FIXME
  name: FIXME
  format: FIXME
  description: FIXME.
  manifests: |
    FIXME
```

Access to:
- `WorkloadName`: string
- `WorkloadSpec`: the Score file itself
- `Location`: `string`
- `ProjectId`: `string`

Template functions:
- `hclescape`: escapes a value for use inside an HCL double-quoted string (`"`, `\`, newlines and control characters). Use it for any user-supplied value in Terraform manifests, e.g. `value = "{{ hclescape $variableValue }}"`. `${...}` is left as-is so Terraform interpolation still works.
