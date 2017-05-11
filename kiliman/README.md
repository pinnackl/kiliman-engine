# Kiliman
## Requirements

 - Golang 1.8+

## Setup

`make setup` downloads golang/dep for vendor management

## Building
 - `make build`
 - `make test`

## Running

 - `./kiliman # Listens on 12345`


## Example Curl

`curl -H "Content-Type: application/json" -X POST -d  {"name" : "titou","email": "ahv@hotmail.fr","offer": "gold" } http://localhost:12345/new-cms`


