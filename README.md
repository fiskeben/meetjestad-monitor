# meetjestad-monitor

A health monitoring service for
[Meet je stad](http://meetjestad.net)
weather stations.
Also check out
[the UI](https://github.com/fiskeben/meetjestad-monitor-ui)
and 
[monitor your own weather station](https://monitoring.meetjescraper.online).

## Features

The service will query a number of sensors at a given frequency
and send an email to the sensor's owner if one of the sensor's
vitals is critical.

The following data is monitored:

* Has the sensor sent any messages in the last six hours?
* Is the sensor's battery voltage running low (<3.26V, configurable)?
* Does the sensor report its location (i.e. do the messages include GPS data)?

Data about sensors and raised alarms is stored in
[Firebase](https://firebase.google.com)
and e-mails are sent with
[Mailgun](https://mailgun.com).

## Building

Run `make meetjestad-monitor` to download dependencies and compile the program for your platform.

The `Makefile` also offers a `dist` target to compile for Linux
which is a common thing to do if you want to deploy the service somewhere.

## Configuration & running

### Config file

Create a YAML file called `config.yaml`
and put it in the same directory as the binary.

The structure of the file is like this:

```yaml
frequency: 1h # duration to wait between checks
mailer:
  domain: yourdomain.com # the domain Mailgun is configured for
  apibase: "https://api.eu.mailgun.net/v3" # for non-US domains
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
If you leave it out, mails are printed to the log.
This can be useful for testing.

### Firestore

To give the service access to Firestore
you need to set up a service account
keep the config file in the same folder as the binary.
Name the file `serviceaccount.json`.

The service requires two collections:

* sensors:
  ```
  sensor_id     string
  threshold     number
  email_address string
  ```
* alarms:
  ```
  gps     time
  offline time
  voltage time
  ```

#### Sensors

The `threshold` field is the value of battery voltage level
that will trigger an alarm.

#### Alarms

All fields are timestamps indicating when the type last
triggered an alarm.

### Running

To run the service just execute `meetjestad-monitor`,
for example `./meetjestad-monitor > service.log 2>&1 &`
to write the logs to a file called `service.log`.
