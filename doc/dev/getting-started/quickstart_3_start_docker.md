# Quickstart step 3: (macOS) Start Docker

## Option A: Docker for Mac

This is the easy way - just launch Docker.app and wait for it to finish loading.

## Option B: docker-machine

The Docker daemon should be running in the background, which you can test by
running `docker ps`. If you're on OS X and using `docker-machine` instead of
Docker for Mac, you may have to run:

```bash
docker-machine start default
eval $(docker-machine env)
```

[< Previous](quickstart_2_initialize_database.md) | [Next >](quickstart_4_clone_repository.md)
