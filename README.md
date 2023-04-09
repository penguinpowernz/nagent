# nagent

An agent that delivers system status in checkmk format via NATS.  It's receiving server 
can modify the incoming data en route, and convert it to JSON using local Debian style scripts.
It also stores/updates a shadow file for each host.  The goal is to have batteries-included
shadow storage for check_mk and check_mk type output.

## Server flow

![image](https://user-images.githubusercontent.com/4642414/209256243-2543f67f-931b-40b1-9c32-89171f8ffb7b.png)

Anything from the agent coming in on the NATS channel is delivered first to the preprocess hook scripts.  These are each run according to their rules, and successful output is then passed to the parsing scripts.

After the parsers have done their work, usually to turn the input into valid JSON to output, that output is passed to the merger/storer which merges the output with any existing JSON documents for the same device name.  This allows agent data to be sent up piecemmeal to be merged into the existing shadows.  The shadows are stored to disk as JSON object and can be accessed by the included HTTP API.

Finally the sink scripts allow the entire resultant merged JSON document to be passed on to any place you want.  You may just need to extend the docker image, or do a feature/pull request to add any extra tools you need to accomplish this task.

Note that the scripts could also be go/rust/etc binaries.  As long as they're executable and adhere to the rules it should work fine.

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

The docker will be the easiest to use.  It comes with ruby, python, `jc`, `yq` and `jq` 
and `gron` preinstalled so you should be able to write any parser you need.

For the sinks `curl` and `mosquitto-clients` are installed as well as some gems like 
`influxdb` and `nats`.  You can extend the docker to add what packages you would find
useful, or do a PR to add them to the Dockerfile in this repository.

```Dockerfile
FROM penguinpower/nagent

RUN apt-get install -y some-other-tool
RUN gem install some-other-lib
```

The volume `/var/lib/nagentd` is exposed so that is where you would put all your scripts.
Using the Docker compose file is recommended, as it includes NATS.

    docker run -it --name nagentd -p "8080:8080" -v "$(pwd)/data:/var/lib/nagentd" penguinpower/nagentd -u nats://172.17.0.1:4222

This assumes you are running NATS on your host.

## Script Hooks

The script hooks all work generally the same:

- all executable files in the folder will be called, in alphanumerical order
- data is passed to STDIN
- modified data is dumped output to STDOUT

They use common exit codes:

- 0: the command was successful, use the output
- 21: ignore the output, use the unmodified input
- 22: discard the entire shadow (pre-hook only)
- *: an error occurred, use the unmodified input

###  Pre scripts (/var/lib/nagentd/pre)

Pre scripts allow you to modify the entire payload before it reaches the parser.  This looks at the entire message not just individual sections.  Incoming data is the raw unformatted check_mk style text via STDIN, e.g:

```
<<<df>>>
tmpfs                 tmpfs     3249480     2508   3246972       1% /run
/dev/mapper/data-root ext4    482351616 46396120 411379908      11% /
/dev/sda1             ext4    491133848       28 466112124       1% /mnt/data
```

Each script should output the same check_mk style text to STDOUT.

### Parser scripts (/var/lib/nagentd/parsers)

Parser scripts receive the raw check_mk style text on STDIN and are expected to process
that text into a JSON object on STDOUT.  The name of the script should refer to the name
of the section it aims to parse.

There is a special case for the script called `_`.  This is a catch-all script and will receive every
section that doesn't have its own script. The name of the section is provided in the envvar `$SECTION`.

The discarding signal (exit code 22) does not work here.  If not parsed, sections will be an array of
strings representing each line of the check_mk section text.

Example script that uses python jc to parse the output: 

    #!/bin/bash
    jc --uname <&0
    exit 0;

Included script to turn the check_mk `cpu` section to a readable load average:

```ruby
#!/usr/bin/env ruby

require 'json'

bits = ARGF.read.split(" ")

puts {
    "1": bits[0].to_f,
    "5": bits[1].to_f,
    "15": bits[2].to_f
}.to_json
```

This is how the included catch-all `_` script works, but can be changed to suit whatever need:

```bash
#!/bin/bash

case $SECTION in
"lsblk")      jc --lsblk <&0 ;;
"df")         jc --df <&0 ;;

# if you pass up YAML in any of your sections you can do this
"myyml1", "myyml2") jc --yaml <&0 ;;

# if you pass up JSON in any of your sections you can do this
"myjsn1", "myjsn2") echo <&0 ;;

# for simple key=value formatted sections you can use the included parser
"keyval1", "keyval2") ./equalsfields ;;

# for simple key:value formatted sections you can use the included parser
"clnkey1", "clnkey2") ./colonfields ;;

*)       exit 1 ;;
esac
```

You can also use symlinks to make multiple sections use the same script if you don't like
to use the catch-all script:

```bash
ln -s equalsfields keyval2
ln -s yaml2json myyml2
```

Included generic parsers:

- `colonfields`
- `equalsfields`
- `xml2json`
- `yaml2json`

### Post (/var/lib/nagentd/post)

These scripts are for modifying the shadow after the sections have all been parsed and 
it has been written to disk.  It receives the the JSON formatted shadow on STDIN and
should dump the modified shadow as JSON on STDOUT.

The discarding signal (exit code 22) does not work here.

### Sinks (/var/lib/nagentd/sinks)

In this folder can go scripts that will receive the fully processed shadow in JSON format
on STDIN. They can be used to transmit the shadow to other sources via webhooks, NATS, MQTT, etc.

Return codes and stdout are ignored (apart from logging) and the `HOSTNAME` envvar is made
available for convenience.

Example webhook:

```bash
#!/bin/bash
curl https://example.com/webhook/xxxxxxx -d@- -H 'Content-Type: application/json' <&0
```

Included NATS script:

```ruby
#!/usr/bin/env ruby
require 'nats'
nats = NATS.connect('nats') # inside the docker-compose the NATS server hostname is 'nats'
nats.publish('shadows.'+ENV['HOSTNAME']+'.updated', ARGF.read)
```

# TODO

- [ ] figure out how to handle multiple sections with the same name (checkmk does this by default sometimes)
- [ ] add logging to the server
- [ ] general refactor to make the code easier to understand
- [ ] don't touch output that is already in JSON
- [ ] add hinting to section names for generic parsers
- [ ] add environment variables for passing secrets to scripts
