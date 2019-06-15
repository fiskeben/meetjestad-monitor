# meetjebatterij

A battery monitoring service for
[Meet je stad](http://meetjestad.net)
weather stations.

## Features

The service will query a number of sensors at a given frequency
and send an email to the sensor's owner if the battery voltage
is below a certain threshold.

The only dependency is an active
[Mailgun](https://mailgun.com)
account.

## Building

Run `make meetjebatterij` to download dependencies and compile the program for your platform.

The `Makefile` also offers a `dist` target to compile for Linux
which is a common thing to do if you want to deploy the service somewhere.

## Configuration & running

### Config file

Create a YAML file called `config.yaml`
and put it in the same directory as the binary.

The structure of the file is like this:

```yaml
service:
  threshold: 3.35 # sensors with a lower voltage than this will trigger alarm
  frequency: 1h # duration to wait between checks
  mailer:
    domain: yourdomain.com # the domain Mailgun is configured for
    apibase: "https://api.eu.mailgun.net/v3" # for non-US domains
subscriptions: # a list of sensor IDs to monitor and e-mail addresses to send alarms to
  - sensorid: 123
    emailaddress: you@example.com
```

The `frequency` is a Go `time.Duration` string.
A duration string is a possibly signed sequence of decimal numbers,
each with optional fraction and a unit suffix,
such as "300ms", "-1.5h" or "2h45m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
_([documentation](https://golang.org/pkg/time/#ParseDuration))_

### Mailgun API key and domain

Currently the service uses Mailgun for sending alerts.
You need an active Mailgun account and it must be set up
with a domain you own.

Once Mailgun is ready you have to configure this service as follows:

* domain name
* API key
* API base URL

All of these values can be found on the domain's page in Mailgun.

The API key must be set as an environment variable called
`MEETJESCRAPER_MAILGUN_API_KEY`.

### Running

To run the service just execute `meetjebatterij`,
for example `./meetjebatterij > service.log 2>&1 &`
to write the logs to a file called `service.log`.
