## `score-cloudrun`

- `--version`|`-v`: version for `score-cloudrun`

## `score-cloudrun init`

Initialise the local state directory and sample Score file.

- `--no-default-profile`: Disable generation of the default workload profile file.

## `score-cloudrun generate`

Run the conversion from Score file to output manifests.

- `--image`|`-i` - An optional container image to use for any container with image == '.'.
- `--location`|`-l` - The Google Cloud Location to deploy to.
- `--output`|`-o` - The output manifests file to write the manifests to (default `manifests.tf`).
- `--output-format` - The output format of the manifests files to write the manifests to (default `terraform`).
- `--override-property` - An optional set of path=key overrides to set or remove.
- `--overrides-file` - `An optional file of Score overrides to merge in.
- `--project-id`|`-p` - The Google Cloud Project ID to deploy to.

Note: you can use the `annotation` below in your Score file to select a particular workload profile:
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: example
  annotations:
    cloudrun.score.dev/workload-profile: default
containers:
  main:
    image: .
```

## `score-cloudrun profiles`

### `list`

The list command will list out the workload profiles. This requires an active score-cloudrun state after `init` has been run. The list of workload profiles will be empty if no profiles are defined.

- `--format`|`-f` - Format of the output: `table` (default), `json`.

## `score-cloudrun provisioners`

### `list`

The list command will list out the resources provisioners. This requires an active `score-cloudrun` state after `init` has been run. The list of resources provisioners will be empty if no provisioners are defined.

- `--format`|`-f` - Format of the output: `table` (default), `json`.