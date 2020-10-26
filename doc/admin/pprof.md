# Generating pprof profiling data

## pprof

[pprof](https://github.com/google/pprof) is a visualization tool for profiling data and all the backend services in a Sourcegraph instance have the ability to generate pprof profiling data.
 
## Exposing the debug port to generate profiling data

Follow the instructions below to generate profiling data. We will use the Sourcegraph frontend and a memory profile as an example (the instructions are easily adapted to any of the Sourcegraph backends and any profiling kind).

### Kubernetes

If you're using the [Kubernetes cluster deployment](https://github.com/sourcegraph/deploy-sourcegraph),  
you need to port-forward 6060 from the frontend pod (if you have more than one, choose one):

```bash script
kubectl get pods
kubectl port-forward sourcegraph-frontend-xxxx 6060:6060
```

### Single-container server deployments

The docker run command for the single-container server needs an additional publish flag to expose the debug port:

```bash script
docker run --publish 7080:7080 --publish 127.0.0.1:3370:3370 --publish 127.0.0.1:6060:6060 --rm --volume ~/.sourcegraph/config:/etc/sourcegraph --volume ~/.sourcegraph/data:/var/opt/sourcegraph sourcegraph/server:3.21.2
```

If Sourcegraph is deployed to a remote server, then access via an SSH tunnel using a tool
such as [sshuttle](https://github.com/sshuttle/sshuttle) is required to establish a secure connection.
To access the remote server using `sshuttle` from your local machine:

```bash script
sshuttle -r user@host 0/0
```

## Generating profiling data

Once the port is reachable, you can trigger a profile dump by sending an HTTP request:
(in the browser or with curl, wget or similar):

```bash script
curl -sK -v http://localhost:6060/debug/pprof/heap > heap.out
```

Once the `heap.out` file has been generated, share it with Sourcegraph support or your account manager for analysis.

## Downloading the binary to use with `go tool pprof`

If you want to use the downloaded profile with `go tool pprof` you need the binary that produced the profile data.

You can use `kubectl` to download it:

```bash script
kubectl cp sourcegraph-frontend-xxxx:/usr/local/bin/frontend frontend-bin
```

Then you can use `go tool pprof`:

```bash script
go tool pprof frontend-bin heap.out
```

## Debug ports

This is a table of Sourcegraph backend debug ports in the two deployment contexts:

|                   | kubernetes | docker |
|-------------------|------------|--------|
| frontend          | 6060       | 6063   |
| gitserver         | 6060       | 6068   |
| searcher          | 6060       | 6069   |
| symbols           | 6060       | 6071   |
| repo-updater      | 6060       | 6074   |
| query-runner      | 6060       | 6067   |
| zoekt-indexserver | 6060       | 6072   |
| zoekt-webserver   | 6060       | 3070   |


## Profiling kinds

- allocs: A sampling of all past memory allocations
- block: Stack traces that led to blocking on synchronization primitives
- cmdline: The command line invocation of the current program
- goroutine: Stack traces of all current goroutines
- heap: A sampling of memory allocations of live objects. You can specify the gc GET parameter to run GC before taking the heap sample.
- mutex: Stack traces of holders of contended mutexes
- profile: CPU profile. You can specify the duration in the seconds GET parameter. After you get the profile file, use the go tool pprof command to investigate the profile.
- threadcreate: Stack traces that led to the creation of new OS threads
trace: A trace of execution of the current program. You can specify the duration in the seconds GET parameter. After you get the trace file, use the go tool trace command to investigate the trace.

Adapting the URL will generate different profile data, for example

```bash script
curl -sK -v http://localhost:6060/debug/pprof/profile > profile.out
```

will generate a CPU profile.
