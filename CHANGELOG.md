# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased

### Added

- Log when something goes wrong with GarbageCollection.

### Fixed

- Correct RBAC rules for the ServiceAccount.

### Changed

- Set up lower resource requests/limits for the Deployment.

## v0.1.1 - 2018-09-02

### Fixed

- Always post StatusCodes to StatusCake, ensuring we error on faulty codes.
- Handle monitor data changes with StatusCake, where an update caused the ID to reset to 0 and create a new monitor.

## v0.1.0 - 2018-09-01

### Added

- Initial Operator configuration
- Support for StatusCake
