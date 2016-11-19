# Squirrel Mac Server

The purpose of this project is to provide the server-side requirements of the [Squirrel.Mac](https://github.com/Squirrel/Squirrel.Mac) auto-update system.  Note that this bears little-to-no relation to the [Squirrel.Windows](https://github.com/Squirrel/Squirrel.Windows) framework, which doesn't require an active server.

Squirrel.Mac is intended to be integrated with an application.  It facilitates the process of checking for, downloading and installing updates.

## Update Process

The flow goes something like this:

* An application initializes Squirrel.Mac, gives it a url to hit to check for updates
* Squirrel requests this url
    * If it gets a "204 No Content" response, it assumes no update is required and reports as much to the application
    * Any other response is parsed as JSON, and the "url" field is extracted
* If an update is required, squirrel downloads the given URL, unzips it, and does various prepatatory steps
* Squirrel notifies the application an update is available
* The application presumably asks for user confirmation, then asks Squirrel to apply the update and then restart the application.

## Existing Server

There is an existing [Squirrel.Server](https://github.com/Squirrel/Squirrel.Server), implemented in ruby.  It wasn't sufficient for our case for the following reasons:

* It's inflexible, and only supports hosting updates for a single application
* It would require us to modify our deployment strategy to include restarting the server

## This Project

This server is dead simple.  It sends a 204 response as appropriate, but otherwise just proxies json files hosted elsewhere  It expects two parameters:

* `url` - the json url to request
* `version` (optional) - the version of the installed application

The server performs the following steps:

* Requests the `url`
* Parses the response as json (handling the various error cases)
* Extracts the `version` field from the json response (which is actually ignored by Squrrel.Mac)
* Compares that against the `version` query parameter
    * If they match, send 204 No Content
    * If not, send the data along

The server makes to attempt to do any caching, since load should be fairly low.

## Running

```
docker build -t squirrel-server-go .
docker run -d -p 3000:3000 squirrel-server-go
```
