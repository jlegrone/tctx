# tctx

[![ci](https://github.com/jlegrone/tctx/actions/workflows/ci.yml/badge.svg)](https://github.com/jlegrone/tctx/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jlegrone/tctx/branch/main/graph/badge.svg?token=jClJfwNTKI)](https://codecov.io/gh/jlegrone/tctx)

`tctx` makes it fast and easy to switch between Temporal clusters when running `tctl` commands.

## Installation

Build from source using [go install](https://golang.org/ref/mod#go-install):

```bash
go install github.com/jlegrone/tctx@latest
```

## Usage

### Add a context

```bash
$ tctx add -c localhost --namespace default --address localhost:7233
Context "localhost" modified.
Active namespace is "default".
```

### Execute a `tctl` command

```bash
$ tctx exec -- tctl cluster health
temporal.api.workflowservice.v1.WorkflowService: SERVING
```

### List contexts

```bash
$ tctx list
NAME          ADDRESS                                                       NAMESPACE    STATUS
localhost     localhost:7233                                                default      active
production    temporal-production.example.com:443                           myapp
staging       temporal-staging.example.com:443                              myapp
```

### Switch contexts

```bash
$ tctx use -c production
Context "production" modified.
Active namespace is "myapp".
```

## Tips

### How it works

`tctx` sets standard Temporal CLI environment variables before executing a subcommand with `tctx exec`.

Any CLI tool (not just `tctl`) can be used in conjunction with `tctx` if it leverages these environment variables.

To view all environment variables set for the current context, run

```bash
tctx exec -- printenv | grep TEMPORAL_CLI
```

By default `tctx exec` uses the active context. The active context is set by the last `tctx use` or `tctx add` command. 
You can override the active context by adding a context flag 

```bash
tctx exec -c <context> -- <command>
```

### Define an alias

Typing `tctx exec -- tctl` is a lot of effort. It's possible to define an alias to make this easier.

```bash
alias tctl="tctx exec -- tctl"
```
