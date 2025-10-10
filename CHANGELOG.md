# Changelog

## [0.11.0](https://github.com/rudderlabs/rudder-iac/compare/v0.10.0...v0.11.0) (2025-10-10)


### Features

* add APIs for event stream tracking plan connections ([#182](https://github.com/rudderlabs/rudder-iac/issues/182)) ([12003ac](https://github.com/rudderlabs/rudder-iac/commit/12003acb1cac03819fc68c00da91e04ac3f385f6))
* add conconcurrency flag to apply and destroy commands ([#186](https://github.com/rudderlabs/rudder-iac/issues/186)) ([5c2258a](https://github.com/rudderlabs/rudder-iac/commit/5c2258aed9db54da3c068b0a7f6d193a19ad4768))
* add CRUD APIs for event stream sources ([#172](https://github.com/rudderlabs/rudder-iac/issues/172)) ([220267f](https://github.com/rudderlabs/rudder-iac/commit/220267fe697ea855f1042edaee38aa6ac1ef57b3))
* add Enabled field to create RETL source requests ([#175](https://github.com/rudderlabs/rudder-iac/issues/175)) ([4232437](https://github.com/rudderlabs/rudder-iac/commit/423243702dce72541c199bf83c0bd3680825f1d4))
* add support for event stream source governance validations ([#190](https://github.com/rudderlabs/rudder-iac/issues/190)) ([71ecef8](https://github.com/rudderlabs/rudder-iac/commit/71ecef8c66537f97de44373494e5dd5833af5b15))
* add support for running operations concurrently ([#167](https://github.com/rudderlabs/rudder-iac/issues/167)) ([cae599c](https://github.com/rudderlabs/rudder-iac/commit/cae599c52a751d2a6035e0a4cb96711dc1376f8a))
* added formatter with ability to format data of specific extension ([#206](https://github.com/rudderlabs/rudder-iac/issues/206)) ([b6ae6ca](https://github.com/rudderlabs/rudder-iac/commit/b6ae6ca815e66b0fbd80e6f73a16af8dbffb752d))
* implement event-stream-source provider ([#181](https://github.com/rudderlabs/rudder-iac/issues/181)) ([58a9da9](https://github.com/rudderlabs/rudder-iac/commit/58a9da99034f63d5e426394295e160083698f27f))
* ruddertyper orchestrating component ([#183](https://github.com/rudderlabs/rudder-iac/issues/183)) ([3000ff7](https://github.com/rudderlabs/rudder-iac/commit/3000ff744133831817b023e403423b9883d625aa))
* separate out the importable resources in the print diff ([#201](https://github.com/rudderlabs/rudder-iac/issues/201)) ([2670ca1](https://github.com/rudderlabs/rudder-iac/commit/2670ca15dba01e04aedb6210081bcec8cad0ce5c))
* use new event stream sources APIs ([#188](https://github.com/rudderlabs/rudder-iac/issues/188)) ([86e7c65](https://github.com/rudderlabs/rudder-iac/commit/86e7c65c17c143c564778dc2623a4b116063fd68))


### Bug Fixes

* enabled field of EventStreamSource is defaulting to false ([#207](https://github.com/rudderlabs/rudder-iac/issues/207)) ([1f84442](https://github.com/rudderlabs/rudder-iac/commit/1f84442fcf7e2ea5130b7bf4d20da28e53990b41))
* incorrect field type used when configuring tracking plan connection ([#212](https://github.com/rudderlabs/rudder-iac/issues/212)) ([74cd314](https://github.com/rudderlabs/rudder-iac/commit/74cd314ba4e0c2b0ef75721a15253b37564fbbc9))
* incorrect response schema used in get event stream sources API ([#209](https://github.com/rudderlabs/rudder-iac/issues/209)) ([dfe7aca](https://github.com/rudderlabs/rudder-iac/commit/dfe7aca4d03b908c07629a498c0f861df649575e))
* move the e2e tests to generic apply and destroy ([#211](https://github.com/rudderlabs/rudder-iac/issues/211)) ([6415b95](https://github.com/rudderlabs/rudder-iac/commit/6415b95d49035ccfaa05de6c9fa017cdc60e3d2a))
* panic when casting tracking plan version to float64 ([#210](https://github.com/rudderlabs/rudder-iac/issues/210)) ([7ed1701](https://github.com/rudderlabs/rudder-iac/commit/7ed1701f43101273e7e65ac819d5ffe8bb1deef7))
* tests failing in main branch ([#187](https://github.com/rudderlabs/rudder-iac/issues/187)) ([a3dca92](https://github.com/rudderlabs/rudder-iac/commit/a3dca923ed9835f6604a2c491d6e06c419b390eb))


### Miscellaneous

* add support to reconstruct state for custom types and properties ([#185](https://github.com/rudderlabs/rudder-iac/issues/185)) ([a6c191c](https://github.com/rudderlabs/rudder-iac/commit/a6c191c0df0bb9e374675f39bd67c5a0073c7d69))
* apiClient - add projectId for events, categories, properties and custom types ([#156](https://github.com/rudderlabs/rudder-iac/issues/156)) ([c407b52](https://github.com/rudderlabs/rudder-iac/commit/c407b52050dd51a37474b3bfe10270ffc1a3986a))
* import apply changes introducing capturing of workspace information and using it in planning operations ([#195](https://github.com/rudderlabs/rudder-iac/issues/195)) ([a7e8e93](https://github.com/rudderlabs/rudder-iac/commit/a7e8e934c4792e43da34592cee9eadc2480f4fb2))
* new workspace importer interface ([#192](https://github.com/rudderlabs/rudder-iac/issues/192)) ([6263dcc](https://github.com/rudderlabs/rudder-iac/commit/6263dcc24b8cc1c51a60fcf3078fcc362cdd578d))
* reconstruct state for tracking plans ([#194](https://github.com/rudderlabs/rudder-iac/issues/194)) ([f8c64e5](https://github.com/rudderlabs/rudder-iac/commit/f8c64e535b95474dea5733363b33e984917af051))

## [0.10.0](https://github.com/rudderlabs/rudder-iac/compare/v0.9.0...v0.10.0) (2025-09-18)


### Features

* add import workspace command to the cli ([#170](https://github.com/rudderlabs/rudder-iac/issues/170)) ([35d7ba4](https://github.com/rudderlabs/rudder-iac/commit/35d7ba431cf11e715eb482be12d6807f901d9f07))
* add RETL source preview functionality ([#154](https://github.com/rudderlabs/rudder-iac/issues/154)) ([6861e00](https://github.com/rudderlabs/rudder-iac/commit/6861e009c22d31bcd93849c315ed91aed76c44b4))
* add sample file manager ([#162](https://github.com/rudderlabs/rudder-iac/issues/162)) ([6b7ce6c](https://github.com/rudderlabs/rudder-iac/commit/6b7ce6c06e678c4529e091aa363b3167ab8cbd2f))
* rudder-typer 2.0, basic infrastructure type aliases ([#132](https://github.com/rudderlabs/rudder-iac/issues/132)) ([ecd5814](https://github.com/rudderlabs/rudder-iac/commit/ecd5814996034bd54eb710458f5b253653221076))
* rudder-typer 2.0, object custom types and properties ([#133](https://github.com/rudderlabs/rudder-iac/issues/133)) ([8ee24b9](https://github.com/rudderlabs/rudder-iac/commit/8ee24b9adde0f47cae0b25abccdd7b8eca4aaf54))
* rudder-typer 2.0, phase3, adds support for event property payloads ([#134](https://github.com/rudderlabs/rudder-iac/issues/134)) ([c496c0b](https://github.com/rudderlabs/rudder-iac/commit/c496c0b97d1cbe4b9721a4cb2ab9424cd9b5bb30))
* rudder-typer 2.0, RudderAnalytics object with event methods ([#135](https://github.com/rudderlabs/rudder-iac/issues/135)) ([9d6fdb0](https://github.com/rudderlabs/rudder-iac/commit/9d6fdb062d78becb85b3415a038e004ff220b7da))


### Bug Fixes

* the API request payload to not omitempty properties when sending event_rule update requests to upstream ([#174](https://github.com/rudderlabs/rudder-iac/issues/174)) ([dad5950](https://github.com/rudderlabs/rudder-iac/commit/dad5950e572d36e75a104d91e4f5c917ab7370ce))


### Miscellaneous

* add new namer package to create a naming abstraction for externalId naming ([#171](https://github.com/rudderlabs/rudder-iac/issues/171)) ([9f40747](https://github.com/rudderlabs/rudder-iac/commit/9f40747ddfae62e731af73433f3bd3f9b3c586b8))
* framework for managing experimental flags ([#173](https://github.com/rudderlabs/rudder-iac/issues/173)) ([d4c5a61](https://github.com/rudderlabs/rudder-iac/commit/d4c5a61f893041f8d108600f71e6dfa11ca99e52))

## [0.9.0](https://github.com/rudderlabs/rudder-iac/compare/v0.8.0...v0.9.0) (2025-08-25)


### Features

* add base support for variants model within trackingplans and customtypes ([#127](https://github.com/rudderlabs/rudder-iac/issues/127)) ([de60651](https://github.com/rudderlabs/rudder-iac/commit/de606510bd825cec1ea4f70f0a4f671c2d36a487))
* add variants to catalog api ([#138](https://github.com/rudderlabs/rudder-iac/issues/138)) ([f24ac67](https://github.com/rudderlabs/rudder-iac/commit/f24ac67f0054516a13a894a9799e0a601025b278))
* **cli:** add import command for RETL SQL models ([#129](https://github.com/rudderlabs/rudder-iac/issues/129)) ([49d7ff0](https://github.com/rudderlabs/rudder-iac/commit/49d7ff0bc155bf85dedf378d961fe3a72bb55318))
* **cli:** implementing import operation for sql models ([#130](https://github.com/rudderlabs/rudder-iac/issues/130)) ([40422bc](https://github.com/rudderlabs/rudder-iac/commit/40422bc31a2b5fa3881266bd9ee2477389efe464))
* **cli:** new command to list retl sources supports only sql models ([#126](https://github.com/rudderlabs/rudder-iac/issues/126)) ([2650b13](https://github.com/rudderlabs/rudder-iac/commit/2650b13d282cd079b9bab65665606c8cec1203cd))
* implement custom differ for custom types to prevent unnecessary updates ([#150](https://github.com/rudderlabs/rudder-iac/issues/150)) ([700633c](https://github.com/rudderlabs/rudder-iac/commit/700633c51f014b1431e4b6ca80568c89f5af36bd))


### Bug Fixes

* make additionalProperties as optional to allow for backward compatible state ([#153](https://github.com/rudderlabs/rudder-iac/issues/153)) ([d110986](https://github.com/rudderlabs/rudder-iac/commit/d110986d80ff77cebe830c194c0b19c48bcb6d78))
* remove unnecessary UnmarshalJSON when handling variantcase and updated the match value validation ([#144](https://github.com/rudderlabs/rudder-iac/issues/144)) ([6fc0592](https://github.com/rudderlabs/rudder-iac/commit/6fc059221b72ab453619ab5bd65084a9c69f3c3d))
* support defining events with the same metadata.name in separate files ([#151](https://github.com/rudderlabs/rudder-iac/issues/151)) ([29a5115](https://github.com/rudderlabs/rudder-iac/commit/29a5115ee1ccf565992f92bdcdbbd673de67ea3a))


### Miscellaneous

* add e2e tests for nested properties ([#146](https://github.com/rudderlabs/rudder-iac/issues/146)) ([8d55f32](https://github.com/rudderlabs/rudder-iac/commit/8d55f325f9661413bc1c4343260bc32876206556))
* add e2e tests for variant support in trackingplan ([#145](https://github.com/rudderlabs/rudder-iac/issues/145)) ([6d9f327](https://github.com/rudderlabs/rudder-iac/commit/6d9f327023378d2f55c9f8ef76058624d11e46e8))
* add nested properties in the CLI ([#136](https://github.com/rudderlabs/rudder-iac/issues/136)) ([95220c3](https://github.com/rudderlabs/rudder-iac/commit/95220c307b654edb46bde5ce88041613b1e9dba3))
* add variants custom type provider ([#142](https://github.com/rudderlabs/rudder-iac/issues/142)) ([fff0487](https://github.com/rudderlabs/rudder-iac/commit/fff04875515d54f1702e90ae8b1dff759ad842b8))

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
