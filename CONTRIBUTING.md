# Contributing

Thanks for your interest in improving the GreenNode vMonitor data source.

## Getting set up

You need Node.js 22 or later (see `.nvmrc`) and the Go toolchain named in
`go.mod`.

```bash
npm install
npm run build     # frontend, into dist/
mage -v           # backend binaries, into dist/
```

Then run a local Grafana with the plugin mounted:

```bash
export VM_CLIENT_ID=... VM_CLIENT_SECRET=...
npm run server
```

That serves <http://localhost:3000> with `dist/` mounted and the data source
provisioned from `provisioning/datasources/datasources.yml`. Build the frontend
and backend at least once first, or `dist/` will be empty.

## Before you open a merge request

```bash
npm run typecheck
npm run lint
go vet ./...
```

`npm run lint:fix` applies the auto-fixable lint and Prettier changes.

## Conventions

- The frontend must be built with the webpack config in `.config/`, and the
  backend with the mage targets from the Grafana plugin Go SDK. Do not edit
  anything inside `.config/` — it is generated and maintained by
  [`@grafana/create-plugin`](https://grafana.com/developers/plugin-tools/).
- Credentials belong in `secureJsonData`; `jsonData` is for non-sensitive
  configuration only. Never commit an access key, and never put one in
  `provisioning/` — that file references `$VM_CLIENT_ID` and
  `$VM_CLIENT_SECRET` from the environment for exactly this reason.
- Changing `src/plugin.json` requires restarting the Grafana server before the
  change takes effect.
- The query editor deliberately mirrors the vMonitor widget builder. When you
  add a query option, keep its name and behaviour aligned with
  [the vMonitor documentation](https://docs.greennode.ai/vmonitor) so that a
  query means the same thing in both places.

## Reporting a bug

Please include the Grafana version, the plugin version, the query (metric name,
statistic, filters — or the log project and search), and the time range. Time
range matters more than you would expect: bucket sizes are derived from it, so
a bug can be visible at one range and invisible at another.
