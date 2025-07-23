# Changelog

## [0.8.0](https://github.com/rudderlabs/rudder-iac/compare/v0.7.0...v0.8.0) (2025-07-22)


### Features

* add e2e tests for the cli ([86a62a9](https://github.com/rudderlabs/rudder-iac/commit/86a62a9f8f927638a86b0a83aad4fb364a4a4f6b))
* api client for accounts api ([#106](https://github.com/rudderlabs/rudder-iac/issues/106)) ([2beb128](https://github.com/rudderlabs/rudder-iac/commit/2beb12828e210fcbfac34cdd8d96d52eb3a27f04))
* api client for retl sources ([#101](https://github.com/rudderlabs/rudder-iac/issues/101)) ([e3f7152](https://github.com/rudderlabs/rudder-iac/commit/e3f71522660229a5a97438be190bf93086ce6e58))
* categories in CLI ([#117](https://github.com/rudderlabs/rudder-iac/issues/117)) ([c3e1608](https://github.com/rudderlabs/rudder-iac/commit/c3e1608c6e1f824ba284f1f2e65714786217dc97))
* **cli:** retl provider for sql models ([#114](https://github.com/rudderlabs/rudder-iac/issues/114)) ([7730391](https://github.com/rudderlabs/rudder-iac/commit/7730391e1a3bdc5f9918d40e547424b7fde78372))
* implement list command that lists workspace accounts ([#107](https://github.com/rudderlabs/rudder-iac/issues/107)) ([f45d38f](https://github.com/rudderlabs/rudder-iac/commit/f45d38f4e2e6b6d98bf8c035d4b365ee73bfd7b5))
* improve error message for duplicate events ([#92](https://github.com/rudderlabs/rudder-iac/issues/92)) ([c4ca81d](https://github.com/rudderlabs/rudder-iac/commit/c4ca81d9801855a3e58344b6d2dde07f8d366ac3))


### Bug Fixes

* resolve panic for missing a matadata name when loading spec ([#90](https://github.com/rudderlabs/rudder-iac/issues/90)) ([8ae456b](https://github.com/rudderlabs/rudder-iac/commit/8ae456b0dedb2a4ed47d949b99a1578fd7e41857))


### Miscellaneous

* add e2e tests for categories ([#124](https://github.com/rudderlabs/rudder-iac/issues/124)) ([1e7611a](https://github.com/rudderlabs/rudder-iac/commit/1e7611a64fc8339131597c9221f4e98f2b443977))
* add README detailing the e2e tests within the README under tests folder ([#111](https://github.com/rudderlabs/rudder-iac/issues/111)) ([db00e49](https://github.com/rudderlabs/rudder-iac/commit/db00e49e217cf81dcf0fc4620ce0d1c4ac2e2270))
* auto-append /v2 to API URLs (RUD-2384) ([#96](https://github.com/rudderlabs/rudder-iac/issues/96)) ([6575755](https://github.com/rudderlabs/rudder-iac/commit/6575755d1d0eadeca4c6356380b41390518a6d56))
* **cli:** show panics in console when debug flag is enabled ([#95](https://github.com/rudderlabs/rudder-iac/issues/95)) ([62ff97e](https://github.com/rudderlabs/rudder-iac/commit/62ff97e44d33d78935f102982e3668b2ca62e4a9))
* extend the entity stores with fetching the entities ([#87](https://github.com/rudderlabs/rudder-iac/issues/87)) ([90066f4](https://github.com/rudderlabs/rudder-iac/commit/90066f46ae844a7ba3e9fb915eadfd5234de7a87))
* merge pkg/provider into internal/providers ([#102](https://github.com/rudderlabs/rudder-iac/issues/102)) ([5801753](https://github.com/rudderlabs/rudder-iac/commit/5801753f43bdc17157e8f7fd416aab8164db0e4d))
* setup e2e test infrastructure ([#94](https://github.com/rudderlabs/rudder-iac/issues/94)) ([62b0b91](https://github.com/rudderlabs/rudder-iac/commit/62b0b918f87d4bea0bc53f9c2a1be3db60d641d2))
* setup the base for e2e tests with binary building and config setup ([#85](https://github.com/rudderlabs/rudder-iac/issues/85)) ([6b36e61](https://github.com/rudderlabs/rudder-iac/commit/6b36e61523325a7f9c930d5ad866093aab2821e5))
* use PR number to build docker images for PRs ([#93](https://github.com/rudderlabs/rudder-iac/issues/93)) ([01e93a9](https://github.com/rudderlabs/rudder-iac/commit/01e93a912057b5097a1b1687b2d50f763f262e6a))

## [0.7.0](https://github.com/rudderlabs/rudder-iac/compare/v0.6.1...v0.7.0) (2025-06-13)


### Features

* add property name validation for leading/trailing whitespace (RUD-2347) ([#80](https://github.com/rudderlabs/rudder-iac/issues/80)) ([70df752](https://github.com/rudderlabs/rudder-iac/commit/70df752bd493ee73853c01324fb72d35613a4dd5))
* refactoring to support multiple providers ([#66](https://github.com/rudderlabs/rudder-iac/issues/66)) ([2ebdc03](https://github.com/rudderlabs/rudder-iac/commit/2ebdc03bf5c393f12341f2822d99a9bb8517b84d))


### Bug Fixes

* 'custom-types' kind added to Data Catalog provider supported kinds ([#81](https://github.com/rudderlabs/rudder-iac/issues/81)) ([dcc6635](https://github.com/rudderlabs/rudder-iac/commit/dcc66351643d06cc05c0f1af376d74746ae394b1))
* panic because of improper initialisation of state ([#79](https://github.com/rudderlabs/rudder-iac/issues/79)) ([c04abe7](https://github.com/rudderlabs/rudder-iac/commit/c04abe7dd167652b3563f5e8567ad1648331ec5e))


### Miscellaneous

* **deps:** bump the go-deps group across 1 directory with 2 updates ([#61](https://github.com/rudderlabs/rudder-iac/issues/61)) ([8fc7abf](https://github.com/rudderlabs/rudder-iac/commit/8fc7abf1a448581b01db2a4ce0c52cd2e9be2059))
* give issue write permission to release please ([#86](https://github.com/rudderlabs/rudder-iac/issues/86)) ([a7caac5](https://github.com/rudderlabs/rudder-iac/commit/a7caac50c3302a79188cda6b7483ad1afaeb3d2e))
* setup release please for automated releases ([#83](https://github.com/rudderlabs/rudder-iac/issues/83)) ([1663968](https://github.com/rudderlabs/rudder-iac/commit/1663968a72643457184930573f9c2ac9d6cf61f7))
