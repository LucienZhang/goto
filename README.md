# goto

[![Go Report Card](https://goreportcard.com/badge/github.com/LucienZhang/goto)](https://goreportcard.com/report/github.com/LucienZhang/goto)

An interactive command-line tool to manage your environments

## Overview

You always need to login to some Linux machine or connect to a MySQL instance. Or you have some directories/environments that you often want to change to.

You can write a script, or create an alias to a sequence of commands, but you just don't want to remember those aliases!

I have many machines to connect to, and I always forget the name of them, so I have to check my script each time!

Let's do something! When you type `goto`, it gives you a list of commands to run, and you can search by the name! You don't need to remember any machine names or aliases!

## Install

```sh
brew install LucienZhang/tap/goto-cli
```

## Config

You need to config your commands in file `~/.goto/.goto.yaml`

```yaml
StartInSearchMode: true # (Optional, default to true, whether start in search mode. press slash (/) to toggle between search mode and normal mode)
Commands:
  - Name: command 1
    Desc: A command that just prints hello # (Optional)
    Color: 244;130;37 # (Optional, this is the RGB color code of your command name. It has to be in form of <r>;<g>;<b>)
    Cmd: echo hello # The command to run
    Shell: bash # (Optional, the shell to run your command, default to your user login shell)
    ExecMode: false # (Optional, default to false, whether run command in exec mode. See details below)
  - Name: ls goto config file
    Cmd: ls ~/.goto/.goto.yaml
```

Then you have this!

![Demo](./docs/demo.gif)

## Execution mode

1. Shell mode. The default mode, when `ExecMode` is `false`. The command is run in a shell, which by default is your user login shell, or the shell you specified in `Shell` field.

2. Exec mode. When `ExecMode` is `true`. The command will be run dirrectly.
