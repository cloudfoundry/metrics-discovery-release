# Get Metrics into Wavefront

## Run Wavefront on Linux VM
Follow instructions at: https://longboard.wavefront.com/integration/telegraf/setup
Mainly this:
```
    bash -c "$(curl -sL https://wavefront.com/install)" \
    -- install \
    --proxy \
    --wavefront-url https://longboard.wavefront.com \
    --api-token <WAVEFRONT_API_TOKEN> \
    --agent \
    --proxy-address localhost \
    --proxy-port 2878
```

## Generate Telegraf Config
The telegraf-config-generator generates a telegraf config from `scrape_targets.json` produced by scrape-config-generator

1. Build telegraf-config-generator
1. Bosh scp the binary to the desired vm running scrape-config-generator
1. Run the telegraf-config-generator
1. View config in `/var/vcap/data/scrape-config-generator/telegraf.toml`

## Run Telegraf

1. Bosh scp the telgraf binary to the same vm
1. `./telegraf --config /var/vcap/data/scrape-config-generator/telegraf.toml`

## Updating config

Each time the `scrape_targets.json` changes the config generator will nee to be re-run
A `SIGHUP` can be sent to Telegraf to reload the `telegraf.toml`
