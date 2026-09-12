---
title: Running with Docker
---

If you already have Docker but prefer not to install `yews`, run the container image `ghcr.io/yewfence/yew-seal:latest` directly. Swap the image tag in the examples as needed.

## File permissions

On Linux/macOS, every command that writes into the mounted `/work` directory (`init`, `encrypt`, `decrypt`, and friends) should pass the host user ID explicitly so the generated files do not end up owned by `root`. On Windows, `--user` can usually be omitted:

```bash
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest --help
```

## Initialization

```bash
docker run --rm -it \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest init
```

## Encrypt / decrypt

```bash
# encrypt
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest encrypt

# decrypt
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest decrypt
```

## Injecting the private key via environment variable

When the private key is injected through an environment variable (CI, for example), mounting a key file is unnecessary:

```bash
docker run --rm \
  -e SOPS_AGE_KEY="$SOPS_AGE_KEY" \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest decrypt
```

## Exporting the key from Infisical

Export the current environment's key with the Infisical CLI on the host, then mount it into the container. See the reference script in [External private key sources](/guide/private-keys#infisical-reference-script); the YewSeal image handles neither Infisical authentication nor remote operations.

## Limitations

`edit` is not recommended under Docker because it depends on a host editor.
