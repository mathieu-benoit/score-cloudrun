# `score-cloudrun`

`score-cloudrun` is a Score implementation of the [Score specification](https://score.dev/) for [Google Cloud Run](https://cloud.google.com/run).

## Demo

Write the following to `score.yaml`:

```bash
make build

./score-cloudrun --version

./score-cloudrun init

./score-cloudrun generate score.yaml
```

```bash
make build-container

docker run --rm -it -v .:/score-cloudrun score-cloudrun:local init

docker run --rm -it -v .:/score-cloudrun score-cloudrun:local generate score.yaml
```