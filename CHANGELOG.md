# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## v0.3.1 - 2019-03-24

### Changed

- We're now using logrus instead of the stdlib log package.

## v0.3.0 - 2019-01-18

### Added

- Added functionality to ensure that SSL checks are enabled with StatusCake

## v0.2.0 - 2018-10-31

### Added

- Log when something goes wrong with GarbageCollection.
- Use a queue based system to sync items.
- Added `/_healthz` endpoint and set up probes.
- Added `/metrics` endpoint for Prometheus.

### Fixed

- Correct RBAC rules for the ServiceAccount.
- Fixed an issue where StatusCake is returning another error for empty data updates.

### Changed

- Set up lower resource requests/limits for the Deployment.
- MonitorTemplate is now namespace scoped instead of cluster scoped.
- The operator now uses cache informers to reconcile state.

### Upgrade Path

- Stop the operator
- Remove the MonitorTemplate and Provider CRD specifications
- Update your own MonitorTemplate and Provider templates to be namespaced
- Apply the new MonitorTemplate and Provider CRD specifications
- Apply the new MonitorTemplate and Provider templates
- Run the new operator

## v0.1.1 - 2018-09-02

### Fixed

- Always post StatusCodes to StatusCake, ensuring we error on faulty codes.
- Handle monitor data changes with StatusCake, where an update caused the ID to reset to 0 and create a new monitor.

## v0.1.0 - 2018-09-01

### Added

- Initial Operator configuration
- Support for StatusCake
