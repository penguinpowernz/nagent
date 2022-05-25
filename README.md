# nagent

An agent that delivers system status in checkmk format via NATS.  It's receiving server converts
the can modify the incoming data en route, and convert it to JSON using local Debian style scripts.
It also stores/updates a shadow file for each host.

## Running the agent

You can throw the output from any command at the stdin and name the 'section' with the `-k` flag.

    uname -a | nagent -k uname

This would result in the following data being sent over NATS:

    <<<uname>>>
    Linux pop-os 5.17.5-76051705-generic #202204271406~1651504840~21.10~63e51bd SMP PREEMPT Mon May 2 15: x86_64 x86_64 x86_64 GNU/Linux

If you have check_mk_agent installed, you can throw its entire output into STDIN:

    check_mk_agent | nagent

I have a backup script, using restic and I dump the snapshots after every backup to a file (`restic --json snapshots > /tmp/snaps.json`).
Into NATS it goes:

    nagent -f /tmp/snaps.json -k restic

In this way, the agent becomes 'cronnable' and you can send data up peicemeal, or particularly taxxing data less often that other data.

Consider the following cron jobs:

```
root 43 * * * * root bash -c 'grep "DRDY ERR" /var/log/kern.log | nagent -k ioerrors'
root 43 * * * * root bash -c 'grep "segfault" /var/log/kern.log | nagent -k segfaults'
root 43 * * * * root bash -c 'grep "oom-kill" /var/log/kern.log | nagent -k oomkills'
root 43 * * * * root bash -c 'grep "Accepted" /var/log/auth.log | nagent -k sshaccepted'
```

## Running the server

Simply run it:

    nagentd

The `-u` flag or the envvar `NATS_URL` will specify the NATS server to connect to.

You can get shadows from the API like: `:8080/shadows/:hostname`

## Running the docker

The docker will be the easiest to use.  It comes with ruby, python, `jc`, `yq` and `jq` and `gron` preinstalled so you should be able to
write any parser you need.

For the sinks `curl` and `mosquitto-clients` are installed as well as some gems like `influxdb` and `nats`.  You can extend the docker to
add what packages you would find useful, or do a PR to add them to the Dockerfile in this repository.

## Script Hooks

###  Pre scripts

Pre scripts allow you to modify the entire payload before it reaches the parser.  This looks at the entire message not just individual sections.

1. incoming data is in JSON format via STDIN
2. all executable files in the folder will be called, in alphanumerical order
3. returned data must be valid JSON
4. return code of 0 means all went OK and STDOUT will be used as the entire message
5. return code of 2 means the entire message should be discarded and further processing stopped
6. any other return code means the script STDOUT will be ignored and the message unmodified

### Parser scripts

Put your scripts in `/var/lib/nagentd/parsers` to do JSON conversions, they should accept the raw data in STDIN and output JSON on STDOUT.
The python `jc` tool can give a quick headstart with this but some scripts are included by default.  For instance, to receive the uname
output sent by the `nagent` in the previous section, you could do the following:

    #!/bin/bash
    # /var/lib/nagentd/parsers/uname

    jc --uname <&0

    exit 0;

As you can surmise the name of the script refers to the name of the section.  If the return code is non-zero then the output will be ignored
and the section will be added to the shadow as a simple array of lines.

There is a special case for the script called `_`.  A script with this name will receive every section that doesn't have its own script and returning 
a non-zero code means the STDOUT will be used for that section.  The envvar `SECTION` will be made available to this script.

    #!/bin/bash

    case $SECTION in
    "lsblk") jc --lsblk ;;
    "df")    jc --df ;;
    *)       exit 1 ;;
    esac


### Sinks

In this folder can go scripts that will receive the fully parsed shadow in JSON format  It can be used to transmit the shadow to other sources
via webhooks etc. Here are the rules:

1. incoming data is in JSON format via STDIN
2. all executable files in the folder will be called, in alphanumerical order
3. STDOUT is ignored
4. return codes are ignored
5. the following envvars are made available to all scripts:
  - HOSTNAME

Example webhook:

    #!/bin/bash
    curl https://example.com/webhook/xxxxxxx -d@- -H 'Content-Type: application/json' <&0

# TODO

- [ ] figure out how to handle multiple sections with the same name (checkmk does this by default sometimes)
- [ ] add logging to the server
- [ ] general refactor to make the code easier to understand