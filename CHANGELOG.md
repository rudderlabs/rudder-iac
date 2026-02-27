# Changelog

## [0.13.0](https://github.com/rudderlabs/rudder-iac/compare/v0.12.1...v0.13.0) (2026-02-27)


### Features

* add base implementation of path indexer ([#343](https://github.com/rudderlabs/rudder-iac/issues/343)) ([badcb88](https://github.com/rudderlabs/rudder-iac/commit/badcb88207de5dcb7dfd79f7fbfe155cb0d1f68b))
* add base semantic validation rules for catalog resource types ([#393](https://github.com/rudderlabs/rudder-iac/issues/393)) ([f05ea1b](https://github.com/rudderlabs/rudder-iac/commit/f05ea1b2170225e5d3d4893d7be4b33033900300))
* add category spec syntax validation ([#377](https://github.com/rudderlabs/rudder-iac/issues/377)) ([7cf27c1](https://github.com/rudderlabs/rudder-iac/commit/7cf27c14630993c227f52a69a36124ee4a996f6a))
* add e2e tests for transformations and libraries ([#354](https://github.com/rudderlabs/rudder-iac/issues/354)) ([5f96003](https://github.com/rudderlabs/rudder-iac/commit/5f96003213308328b495ac658c375c0fd75f4933))
* add event spec syntax validation ([#376](https://github.com/rudderlabs/rudder-iac/issues/376)) ([4e9c4c1](https://github.com/rudderlabs/rudder-iac/commit/4e9c4c1aee80e2fc93b7c11361989fb6ad89a004))
* add event stream validation rules ([#404](https://github.com/rudderlabs/rudder-iac/issues/404)) ([9e3d263](https://github.com/rudderlabs/rudder-iac/commit/9e3d263feab22c525099bc45414e8c48dc1f312b))
* add project rule for the duplicate local id validation ([#397](https://github.com/rudderlabs/rudder-iac/issues/397)) ([0383e65](https://github.com/rudderlabs/rudder-iac/commit/0383e65e9cfbfe251d43d2c2597a58f71747a776))
* add semantic validation rules for trackingplans and customtypes ([#395](https://github.com/rudderlabs/rudder-iac/issues/395)) ([9fd1823](https://github.com/rudderlabs/rudder-iac/commit/9fd182311e8e551ec4cff489dfbba7e970db68bb))
* add spec validation rules ([#362](https://github.com/rudderlabs/rudder-iac/issues/362)) ([a9feda6](https://github.com/rudderlabs/rudder-iac/commit/a9feda6df29b11f88e1368c038efdc504339bd9b))
* add template for pull request ([#416](https://github.com/rudderlabs/rudder-iac/issues/416)) ([8859e83](https://github.com/rudderlabs/rudder-iac/commit/8859e83f2e4951ca6c3fe0f892036abed86aba55))
* add version-aware rule contract and wire project syntax rules to supported versions ([#415](https://github.com/rudderlabs/rudder-iac/issues/415)) ([2009df3](https://github.com/rudderlabs/rudder-iac/commit/2009df30cfc68bf0f28db1a789b6f4abf002f9df))
* added relationship support to data graph provider ([#361](https://github.com/rudderlabs/rudder-iac/issues/361)) ([e7384d0](https://github.com/rudderlabs/rudder-iac/commit/e7384d06a0c589e2a1ac1553eaddb130167babd3))
* added support for rules and rules registry ([#344](https://github.com/rudderlabs/rudder-iac/issues/344)) ([613530b](https://github.com/rudderlabs/rudder-iac/commit/613530b2f525321b8da31ff4690cc06ae8200214))
* added support for spec syntax validation on custom types ([#381](https://github.com/rudderlabs/rudder-iac/issues/381)) ([f7115a5](https://github.com/rudderlabs/rudder-iac/commit/f7115a541330c2101423a151b7cbc3f1c63ade9d))
* added support for user defined validation functions ([#373](https://github.com/rudderlabs/rudder-iac/issues/373)) ([f88072f](https://github.com/rudderlabs/rudder-iac/commit/f88072f38fd675d7eaccce60a49c6cafec2a503c))
* added tracking plan spec syntax validation ([#382](https://github.com/rudderlabs/rudder-iac/issues/382)) ([5fe2cb7](https://github.com/rudderlabs/rudder-iac/commit/5fe2cb7930513c2898ddd3806bbdda557a29176b))
* added unniqueness check on the event and category resource from the graph ([#394](https://github.com/rudderlabs/rudder-iac/issues/394)) ([97d2264](https://github.com/rudderlabs/rudder-iac/commit/97d2264682200a9b263174d34f42be1595f5a1f1))
* **api:** initial data-graph API client ([#351](https://github.com/rudderlabs/rudder-iac/issues/351)) ([902356a](https://github.com/rudderlabs/rudder-iac/commit/902356ab021ad20acf21791028147dd40dd418cd))
* consolidate syncer integration to composite provider ([#353](https://github.com/rudderlabs/rudder-iac/issues/353)) ([14e698c](https://github.com/rudderlabs/rudder-iac/commit/14e698cd1e39548dd02559b37a4c745dea8af3a7))
* data graph provider foundation ([#359](https://github.com/rudderlabs/rudder-iac/issues/359)) ([0aa44ba](https://github.com/rudderlabs/rudder-iac/commit/0aa44ba710016e75f50fb412672ad39960d00565))
* implement handlers for transformation and library ([#339](https://github.com/rudderlabs/rudder-iac/issues/339)) ([e9de343](https://github.com/rudderlabs/rudder-iac/commit/e9de3430c849db83b82b219e8265d9bf48d64c7e))
* import flow for transformations and libraries ([#358](https://github.com/rudderlabs/rudder-iac/issues/358)) ([0512f99](https://github.com/rudderlabs/rudder-iac/commit/0512f99e0625eed151ee314c0844358ed6860a29))
* initial setup for the spec migrator ([#327](https://github.com/rudderlabs/rudder-iac/issues/327)) ([6e940e4](https://github.com/rudderlabs/rudder-iac/commit/6e940e44eac0ac0491791cab047d253628c39145))
* javascript code parser for transformations ([#337](https://github.com/rudderlabs/rudder-iac/issues/337)) ([ed132e3](https://github.com/rudderlabs/rudder-iac/commit/ed132e3a3ddfe10f1b6ab7aed84140ddc238ca3c))
* model support in data graph provider ([#360](https://github.com/rudderlabs/rudder-iac/issues/360)) ([9c2b7ea](https://github.com/rudderlabs/rudder-iac/commit/9c2b7eaa3033cb832a7cb984ca33f7cf78a0f03f))
* new generic provider and example provider to demonstrate the new framework ([#310](https://github.com/rudderlabs/rudder-iac/issues/310)) ([38f2176](https://github.com/rudderlabs/rudder-iac/commit/38f2176acec032e75c5848ee1539df1e20020fe9))
* optimise path indexer to be present on the rawspecs ([#406](https://github.com/rudderlabs/rudder-iac/issues/406)) ([c20aec5](https://github.com/rudderlabs/rudder-iac/commit/c20aec5b9efd775bb1d503080ae2a2d19d37da7a))
* provider implementation for transformations cli ([#352](https://github.com/rudderlabs/rudder-iac/issues/352)) ([49947de](https://github.com/rudderlabs/rudder-iac/commit/49947def89e03daf168e28c295c273e40ccd6e9d))
* python parser implementation ([#389](https://github.com/rudderlabs/rudder-iac/issues/389)) ([a058d93](https://github.com/rudderlabs/rudder-iac/commit/a058d93f3bce938f9f102ee0fe3b87542d7cc85f))
* test command implementation structure ([#371](https://github.com/rudderlabs/rudder-iac/issues/371)) ([f5d12ff](https://github.com/rudderlabs/rudder-iac/commit/f5d12fff38cf89afe4f52e8d8ebab5743b4b8479))
* test command orchestrator ([#372](https://github.com/rudderlabs/rudder-iac/issues/372)) ([8273e84](https://github.com/rudderlabs/rudder-iac/commit/8273e84e62612dc751b7552ac5b55dda997008d2))
* validation engine ([#357](https://github.com/rudderlabs/rudder-iac/issues/357)) ([25458a8](https://github.com/rudderlabs/rudder-iac/commit/25458a8fbfdaebf36e53da06056a2153d68ea44b))


### Bug Fixes

* added validation rules which were present in the previous validation but not in new ([#402](https://github.com/rudderlabs/rudder-iac/issues/402)) ([887ff3d](https://github.com/rudderlabs/rudder-iac/commit/887ff3dc3f494930a53ac74cc0a448105eec4d9c))
* dereference PropertyRefs in update and delete operations ([#399](https://github.com/rudderlabs/rudder-iac/issues/399)) ([764c1de](https://github.com/rudderlabs/rudder-iac/commit/764c1de07ffaaaeb0c6062112478dae120823666))
* fix default value for additionalProps to false for an array type property without itemTypes ([#400](https://github.com/rudderlabs/rudder-iac/issues/400)) ([1dc7dc0](https://github.com/rudderlabs/rudder-iac/commit/1dc7dc02cbf8c5a6cfe2b0e247f6ce1dcba4a403))
* fix QA issues ([#392](https://github.com/rudderlabs/rudder-iac/issues/392)) ([72d841d](https://github.com/rudderlabs/rudder-iac/commit/72d841d30d1e9786707448f7f7a64343e037f539))
* include dependencies in remote resource state ([#388](https://github.com/rudderlabs/rudder-iac/issues/388)) ([f0741e9](https://github.com/rudderlabs/rudder-iac/commit/f0741e98148dae22f2d24a68ca25eec63117d362))
* **migrate-spec:** dont add empty fields to the migrated YAMLs ([#396](https://github.com/rudderlabs/rudder-iac/issues/396)) ([82ced3d](https://github.com/rudderlabs/rudder-iac/commit/82ced3d6943dd42aeb06af821faf21d8a3d4ce64))
* remoteID resolution for importable resources ([#414](https://github.com/rudderlabs/rudder-iac/issues/414)) ([d7ddc47](https://github.com/rudderlabs/rudder-iac/commit/d7ddc4788c7be83f62e4169bf52d94d2184ac3e7))
* revamp the config validation for custom types and property ([#386](https://github.com/rudderlabs/rudder-iac/issues/386)) ([7ff3e91](https://github.com/rudderlabs/rudder-iac/commit/7ff3e916d27afb9e16b6924d0b5eb56c6829d7da))
* transformation delete before publish using deferred deletes ([#407](https://github.com/rudderlabs/rudder-iac/issues/407)) ([c8d6783](https://github.com/rudderlabs/rudder-iac/commit/c8d67832165411c77ea2d014be92dc5836281b5a))
* validations based on QA feedback ([#409](https://github.com/rudderlabs/rudder-iac/issues/409)) ([144ad32](https://github.com/rudderlabs/rudder-iac/commit/144ad32782cc0ae9f1d30c9057f7a0f97ac2c430))


### Miscellaneous

* add batch test api to transformations client ([#368](https://github.com/rudderlabs/rudder-iac/issues/368)) ([cb4592e](https://github.com/rudderlabs/rudder-iac/commit/cb4592ea80e79effffba06492b5c4870a307c822))
* add CLAUDE.md for the repository as a starting point ([#330](https://github.com/rudderlabs/rudder-iac/issues/330)) ([723ccbb](https://github.com/rudderlabs/rudder-iac/commit/723ccbb0216335ed5725b770ea0abc5f8d01bad1))
* add models for transformation and library ([#338](https://github.com/rudderlabs/rudder-iac/issues/338)) ([b951640](https://github.com/rudderlabs/rudder-iac/commit/b95164073b5d9922b531e1713a0fa5dd5e7cbaf9))
* add specs for transformation and library ([#336](https://github.com/rudderlabs/rudder-iac/issues/336)) ([91e5bdf](https://github.com/rudderlabs/rudder-iac/commit/91e5bdf80c13200fa256fe53c92d3ffdcbbd2c1a))
* add support to apply common migrations to all specs ([#342](https://github.com/rudderlabs/rudder-iac/issues/342)) ([c9d019b](https://github.com/rudderlabs/rudder-iac/commit/c9d019b0b35faf5612beec61911e72c9721c149a))
* capture error message too from api errors ([#384](https://github.com/rudderlabs/rudder-iac/issues/384)) ([1208166](https://github.com/rudderlabs/rudder-iac/commit/1208166d3a3aaed0389d6b9d368259d522a2b337))
* change kind for tracking plans from tp to tracking-plan + include events and category specs while migrating ([#379](https://github.com/rudderlabs/rudder-iac/issues/379)) ([c2a5836](https://github.com/rudderlabs/rudder-iac/commit/c2a583633380fb54919a53b37a0e48ef0855bf6c))
* convert path based references to URN based references ([#366](https://github.com/rudderlabs/rudder-iac/issues/366)) ([f88e83f](https://github.com/rudderlabs/rudder-iac/commit/f88e83f78abd943931dcf0b4a7861d28438dcb31))
* **custom-types:** convert variant default from array of properties to object ([#370](https://github.com/rudderlabs/rudder-iac/issues/370)) ([229c31a](https://github.com/rudderlabs/rudder-iac/commit/229c31a6b6ce01d5f2c23249402d8cdf703d838b))
* **custom-types:** rename $ref field to property ([#369](https://github.com/rudderlabs/rudder-iac/issues/369)) ([320a2a1](https://github.com/rudderlabs/rudder-iac/commit/320a2a1d46cb0a4e84dd8577aa4c4a81d0dc0e6d))
* display test suite results for transformation tests ([#391](https://github.com/rudderlabs/rudder-iac/issues/391)) ([022715d](https://github.com/rudderlabs/rudder-iac/commit/022715dde690e2d9dff7520ed702c30d9578d782))
* enable gosec security linter ([#403](https://github.com/rudderlabs/rudder-iac/issues/403)) ([6c28bc9](https://github.com/rudderlabs/rudder-iac/commit/6c28bc93f7ef1b2a65459f26c677148fdbd9085f))
* experimental flag to support transformations ([#390](https://github.com/rudderlabs/rudder-iac/issues/390)) ([e7249a7](https://github.com/rudderlabs/rudder-iac/commit/e7249a7e2a524a47cb4ca9304e0f88a23f9d6ceb))
* **import:** create references in the new format ([#380](https://github.com/rudderlabs/rudder-iac/issues/380)) ([e327442](https://github.com/rudderlabs/rudder-iac/commit/e327442fb13e25b6584cbac9636392e3620514dd))
* include spec directory in model ([#367](https://github.com/rudderlabs/rudder-iac/issues/367)) ([fdf0ac1](https://github.com/rudderlabs/rudder-iac/commit/fdf0ac1e0776f755f6bdb02161f0301e6573b088))
* integrate test flow in apply cmd ([#411](https://github.com/rudderlabs/rudder-iac/issues/411)) ([e36fd16](https://github.com/rudderlabs/rudder-iac/commit/e36fd167dde172b9b45e1687fa84627cf84ba530))
* **properties:** read array itemType/itemTypes via dedicated field ([#363](https://github.com/rudderlabs/rudder-iac/issues/363)) ([6c82f1f](https://github.com/rudderlabs/rudder-iac/commit/6c82f1fabb6bcf50e4cdbf4033d882a81dcd8c0c))
* **properties:** split type field into type and types ([#348](https://github.com/rudderlabs/rudder-iac/issues/348)) ([5546c83](https://github.com/rudderlabs/rudder-iac/commit/5546c8373b8b882b926c3697798a19be0d30868d))
* **propertySpec:** rename propConfig to config and transform config fields to snakecase ([#346](https://github.com/rudderlabs/rudder-iac/issues/346)) ([2248159](https://github.com/rudderlabs/rudder-iac/commit/22481596651cdb27b4d60f39a3666781e98614db))
* test command integration to cli root ([#408](https://github.com/rudderlabs/rudder-iac/issues/408)) ([708c626](https://github.com/rudderlabs/rudder-iac/commit/708c62640af7ee5d346c2bbbc56e7f49e97e4098))
* text formatter to import transformations code ([#355](https://github.com/rudderlabs/rudder-iac/issues/355)) ([da46162](https://github.com/rudderlabs/rudder-iac/commit/da46162b4a9d4ff825b26e85988522c222d7267a))
* **tracking-plans:** migrate $ref fields to semantically named reference fields ([#378](https://github.com/rudderlabs/rudder-iac/issues/378)) ([804128d](https://github.com/rudderlabs/rudder-iac/commit/804128d343bb26a133c660674ad1f34c7cdcea15))
* **tracking-plans:** restructure rules.event to a direct reference field + add nested fields to rules ([#375](https://github.com/rudderlabs/rudder-iac/issues/375)) ([8b1bdff](https://github.com/rudderlabs/rudder-iac/commit/8b1bdff88c0e01df147ac366255b20a986ea4bbb))
* transformations cli client ([#335](https://github.com/rudderlabs/rudder-iac/issues/335)) ([a0c0850](https://github.com/rudderlabs/rudder-iac/commit/a0c0850ea731b15227c4134b94b2f94c2729bef0))

## [0.12.1](https://github.com/rudderlabs/rudder-iac/compare/v0.12.0...v0.12.1) (2025-12-18)


### Miscellaneous

* additionalProperties for custom type props should default to false ([#323](https://github.com/rudderlabs/rudder-iac/issues/323)) ([698f30d](https://github.com/rudderlabs/rudder-iac/commit/698f30dd0d78fdadb0a215cb91325f070cb81bb0))
* take additionalProps into account when diffing TPE props ([#321](https://github.com/rudderlabs/rudder-iac/issues/321)) ([9d256c2](https://github.com/rudderlabs/rudder-iac/commit/9d256c23cfbfcb77fd4d9379153dcb86c295af21))

## [0.12.0](https://github.com/rudderlabs/rudder-iac/compare/v0.11.2...v0.12.0) (2025-12-15)


### Features

* batched trackingplan event update requests ([#308](https://github.com/rudderlabs/rudder-iac/issues/308)) ([b559f7e](https://github.com/rudderlabs/rudder-iac/commit/b559f7e469f8833e0a5c67b6945b391ea7a1e934))
* **cli:** add strict validation to reject unknown fields in YAML specs ([#315](https://github.com/rudderlabs/rudder-iac/issues/315)) ([29349e1](https://github.com/rudderlabs/rudder-iac/commit/29349e138706ac1e80818c240e16fca42130b942))
* experimental public package to expose project loading ([#309](https://github.com/rudderlabs/rudder-iac/issues/309)) ([640b323](https://github.com/rudderlabs/rudder-iac/commit/640b32340b71d94e80fdae12d3ed1d5473c187f3))
* improved apply command with concurrent execution ([#298](https://github.com/rudderlabs/rudder-iac/issues/298)) ([e917a2f](https://github.com/rudderlabs/rudder-iac/commit/e917a2f742591867f4995a39dc4ea44939dc429e))
* remove the constraints on the ordering of the event states ([#292](https://github.com/rudderlabs/rudder-iac/issues/292)) ([6343b6c](https://github.com/rudderlabs/rudder-iac/commit/6343b6c135f208e69e5a827c865ff95139ee1769))
* track CI/CD vs local execution context  ([#319](https://github.com/rudderlabs/rudder-iac/issues/319)) ([024be1d](https://github.com/rudderlabs/rudder-iac/commit/024be1d0023e910310d4cb29eefecb656a99f357))
* use rebuildSchemas=false to fetch latest mappings ([#286](https://github.com/rudderlabs/rudder-iac/issues/286)) ([b493933](https://github.com/rudderlabs/rudder-iac/commit/b493933bddc690525879ece3f0044c775fe7662e))
* **validation:** detect circular dependencies between resources ([#318](https://github.com/rudderlabs/rudder-iac/issues/318)) ([df81c70](https://github.com/rudderlabs/rudder-iac/commit/df81c701cc930743f88cc2e478d09f28b24c3bc3))


### Bug Fixes

* add tests for skipping sources with no external state in load resources from remote ([#296](https://github.com/rudderlabs/rudder-iac/issues/296)) ([f902543](https://github.com/rudderlabs/rudder-iac/commit/f9025437856ea36f2879a6c94c76bd39986e3218))


### Miscellaneous

* add test for the tasker framework ([#311](https://github.com/rudderlabs/rudder-iac/issues/311)) ([8c12c68](https://github.com/rudderlabs/rudder-iac/commit/8c12c6805ef597d6ca07c6efabe9837ddfd42898))
* allow setting additonalProperties for nested properties ([#297](https://github.com/rudderlabs/rudder-iac/issues/297)) ([d3b5c97](https://github.com/rudderlabs/rudder-iac/commit/d3b5c972d452f396613523e0cc403b0ff71fa55f))
* clean up provider interfaces ([#306](https://github.com/rudderlabs/rudder-iac/issues/306)) ([a375541](https://github.com/rudderlabs/rudder-iac/commit/a3755415e53c511af52f3fe0ecc213924709afd0))
* enable concurrent syncs e2e test ([#313](https://github.com/rudderlabs/rudder-iac/issues/313)) ([6bf74c3](https://github.com/rudderlabs/rudder-iac/commit/6bf74c321691021a16893a5ba2a7279239c0dfa0))
* moved resources package outside syncer  ([#303](https://github.com/rudderlabs/rudder-iac/issues/303)) ([be4b08d](https://github.com/rudderlabs/rudder-iac/commit/be4b08d64af88fd8084d67080d86f568ef161876))
* remove deprecated read/load/put/delete State operations ([#301](https://github.com/rudderlabs/rudder-iac/issues/301)) ([3dcda73](https://github.com/rudderlabs/rudder-iac/commit/3dcda73475ea4051885a84d5cd9af2d992de08f5))
* **typer:** refactor kotlin tests to use the actual SDK for validations ([#253](https://github.com/rudderlabs/rudder-iac/issues/253)) ([332e9a2](https://github.com/rudderlabs/rudder-iac/commit/332e9a24d847299391aa8129726fe7ecfc29a431))
* unify import implementations of bulk workspace import and single resource retl import ([#307](https://github.com/rudderlabs/rudder-iac/issues/307)) ([6d5842a](https://github.com/rudderlabs/rudder-iac/commit/6d5842aa792120217c379eb6fea53bb542deba8f))

## [0.11.2](https://github.com/rudderlabs/rudder-iac/compare/v0.11.1...v0.11.2) (2025-11-18)


### Bug Fixes

* filter the list of sources which have no externalID set ([#294](https://github.com/rudderlabs/rudder-iac/issues/294)) ([da8b538](https://github.com/rudderlabs/rudder-iac/commit/da8b53891b3e7a3f49cd72c361f6cd51d84d3111))

## [0.11.1](https://github.com/rudderlabs/rudder-iac/compare/v0.11.0...v0.11.1) (2025-11-14)


### Bug Fixes

* use 1.24.0 go version when running releaser ([#289](https://github.com/rudderlabs/rudder-iac/issues/289)) ([edba33c](https://github.com/rudderlabs/rudder-iac/commit/edba33c2126205c153443db588be47c93d60194f))

## [0.11.0](https://github.com/rudderlabs/rudder-iac/compare/v0.10.0...v0.11.0) (2025-11-14)


### Features

* add ability to import custom types from upstream ([#213](https://github.com/rudderlabs/rudder-iac/issues/213)) ([2d2e4ec](https://github.com/rudderlabs/rudder-iac/commit/2d2e4ecfb04c4b72c9edcd341fe947fcd5aad0c9))
* add APIs for event stream tracking plan connections ([#182](https://github.com/rudderlabs/rudder-iac/issues/182)) ([12003ac](https://github.com/rudderlabs/rudder-iac/commit/12003acb1cac03819fc68c00da91e04ac3f385f6))
* add conconcurrency flag to apply and destroy commands ([#186](https://github.com/rudderlabs/rudder-iac/issues/186)) ([5c2258a](https://github.com/rudderlabs/rudder-iac/commit/5c2258aed9db54da3c068b0a7f6d193a19ad4768))
* add CRUD APIs for event stream sources ([#172](https://github.com/rudderlabs/rudder-iac/issues/172)) ([220267f](https://github.com/rudderlabs/rudder-iac/commit/220267fe697ea855f1042edaee38aa6ac1ef57b3))
* add Enabled field to create RETL source requests ([#175](https://github.com/rudderlabs/rudder-iac/issues/175)) ([4232437](https://github.com/rudderlabs/rudder-iac/commit/423243702dce72541c199bf83c0bd3680825f1d4))
* add support for event stream source governance validations ([#190](https://github.com/rudderlabs/rudder-iac/issues/190)) ([71ecef8](https://github.com/rudderlabs/rudder-iac/commit/71ecef8c66537f97de44373494e5dd5833af5b15))
* add support for running operations concurrently ([#167](https://github.com/rudderlabs/rudder-iac/issues/167)) ([cae599c](https://github.com/rudderlabs/rudder-iac/commit/cae599c52a751d2a6035e0a4cb96711dc1376f8a))
* add tests for the import handler for the data catalog entities ([#244](https://github.com/rudderlabs/rudder-iac/issues/244)) ([756e609](https://github.com/rudderlabs/rudder-iac/commit/756e609fae6841e00ae139ed9314bc6a841bdb7e))
* add validations for invalid tracking plan references in event stream sources ([#224](https://github.com/rudderlabs/rudder-iac/issues/224)) ([a36c0fb](https://github.com/rudderlabs/rudder-iac/commit/a36c0fb652b21dcc82afae210676532014f3608f))
* added ability to validate spec in terms of import metadata at global level ([#248](https://github.com/rudderlabs/rudder-iac/issues/248)) ([8f2cc38](https://github.com/rudderlabs/rudder-iac/commit/8f2cc38ef387c3515fb7ed041868c5776c09a77d))
* added base support for importing categories in the system ([#215](https://github.com/rudderlabs/rudder-iac/issues/215)) ([21ea8cf](https://github.com/rudderlabs/rudder-iac/commit/21ea8cf4021d6a64b8a082068d99404fc1ebe72c))
* added formatter with ability to format data of specific extension ([#206](https://github.com/rudderlabs/rudder-iac/issues/206)) ([b6ae6ca](https://github.com/rudderlabs/rudder-iac/commit/b6ae6ca815e66b0fbd80e6f73a16af8dbffb752d))
* added spinner for the import flow to have a visual feedback when importing ([#229](https://github.com/rudderlabs/rudder-iac/issues/229)) ([a87a8a7](https://github.com/rudderlabs/rudder-iac/commit/a87a8a7824e4f26e15a36996408b1e9750e37210))
* added support for listoptions to filter the entities when fetching from upstream ([#274](https://github.com/rudderlabs/rudder-iac/issues/274)) ([db7de88](https://github.com/rudderlabs/rudder-iac/commit/db7de88064c20e8b9eb5244685bc0fbcc297b403))
* error out on importing if the directory for import already exists ([#220](https://github.com/rudderlabs/rudder-iac/issues/220)) ([4b14ea3](https://github.com/rudderlabs/rudder-iac/commit/4b14ea3970a935e881bc60138be3465f43c76a2d))
* implement event-stream-source provider ([#181](https://github.com/rudderlabs/rudder-iac/issues/181)) ([58a9da9](https://github.com/rudderlabs/rudder-iac/commit/58a9da99034f63d5e426394295e160083698f27f))
* implement import operation for event stream sources ([#241](https://github.com/rudderlabs/rudder-iac/issues/241)) ([b5f5dd7](https://github.com/rudderlabs/rudder-iac/commit/b5f5dd7438c7517e0cbb1fc62205b7f461e38fe4))
* implement new import flow for sql models ([#249](https://github.com/rudderlabs/rudder-iac/issues/249)) ([d9e483a](https://github.com/rudderlabs/rudder-iac/commit/d9e483a6a7a277e09f2037fb4f436db30de05a20))
* import functionality for event stream sources ([#218](https://github.com/rudderlabs/rudder-iac/issues/218)) ([c93cbc3](https://github.com/rudderlabs/rudder-iac/commit/c93cbc3b89a0cfa8de4439372bfdb292e136645c))
* import tracking plans ([#221](https://github.com/rudderlabs/rudder-iac/issues/221)) ([e7ef563](https://github.com/rudderlabs/rudder-iac/commit/e7ef563c1a4ab5d1cda9b498343fcae974af0b31))
* parallelise the startup of the cli and import ([#278](https://github.com/rudderlabs/rudder-iac/issues/278)) ([b178a5a](https://github.com/rudderlabs/rudder-iac/commit/b178a5a655ca885b316924b18f780554d091e51e))
* prevent importing workspace when detected changes not synced ([#219](https://github.com/rudderlabs/rudder-iac/issues/219)) ([9650787](https://github.com/rudderlabs/rudder-iac/commit/96507876703258b044df4e713960c4b936405a52))
* **retl:** add external ID support to RETL sources ([#227](https://github.com/rudderlabs/rudder-iac/issues/227)) ([c787ed3](https://github.com/rudderlabs/rudder-iac/commit/c787ed362cdb697fabcdb32847230580e66dda25))
* ruddertyper orchestrating component ([#183](https://github.com/rudderlabs/rudder-iac/issues/183)) ([3000ff7](https://github.com/rudderlabs/rudder-iac/commit/3000ff744133831817b023e403423b9883d625aa))
* separate out the importable resources in the print diff ([#201](https://github.com/rudderlabs/rudder-iac/issues/201)) ([2670ca1](https://github.com/rudderlabs/rudder-iac/commit/2670ca15dba01e04aedb6210081bcec8cad0ce5c))
* show listing table for all tracking plans ([#225](https://github.com/rudderlabs/rudder-iac/issues/225)) ([f1c2b3f](https://github.com/rudderlabs/rudder-iac/commit/f1c2b3ff46e3d891176b3f2f18ae120dcb3531d3))
* turn on statelessCLI by default ([#256](https://github.com/rudderlabs/rudder-iac/issues/256)) ([83b8fad](https://github.com/rudderlabs/rudder-iac/commit/83b8fadc575d8d13b95dd81874ab5bfa874aa87b))
* **type:** improved handling of Kotlin keywords ([#232](https://github.com/rudderlabs/rudder-iac/issues/232)) ([d339bb4](https://github.com/rudderlabs/rudder-iac/commit/d339bb44b61f9eb726563a315488c5f1d5ce98a2))
* **typer:** add enum & arrays support for Kotlin generator ([#196](https://github.com/rudderlabs/rudder-iac/issues/196)) ([8e8c7d5](https://github.com/rudderlabs/rudder-iac/commit/8e8c7d5a745ec28fba50d617629b0d44819d6efe))
* **typer:** add support for properties and arrays with multiple types ([#208](https://github.com/rudderlabs/rudder-iac/issues/208)) ([caad086](https://github.com/rudderlabs/rudder-iac/commit/caad086a3ed2a1bbd5cae75f27cc0d5fe1814ed2))
* **typer:** additional check for plans with multiple variants (not supported) ([#233](https://github.com/rudderlabs/rudder-iac/issues/233)) ([f02ad6e](https://github.com/rudderlabs/rudder-iac/commit/f02ad6e71965a1ffe0e603fab696be13fda090a7))
* **typer:** discriminator default value for non string types ([#234](https://github.com/rudderlabs/rudder-iac/issues/234)) ([1a03efe](https://github.com/rudderlabs/rudder-iac/commit/1a03efec43b84fd9739d3db491cb89562539758f))
* **typer:** extended Kotlin custom types support ([#203](https://github.com/rudderlabs/rudder-iac/issues/203)) ([4d96278](https://github.com/rudderlabs/rudder-iac/commit/4d96278455249127a2ba5aa4354646585aff1a47))
* **typer:** improved kotlin docs ([#269](https://github.com/rudderlabs/rudder-iac/issues/269)) ([8890c03](https://github.com/rudderlabs/rudder-iac/commit/8890c03ac85922ffe82739455294e43dfa0d45ab))
* **typer:** improved kotlin KDocs in generated code ([#261](https://github.com/rudderlabs/rudder-iac/issues/261)) ([1d78bc4](https://github.com/rudderlabs/rudder-iac/commit/1d78bc4b555c7bae10cd9bbac13cf2cd61e7439e))
* **typer:** introduced platform options framework to support custom package names for kotlin ([#258](https://github.com/rudderlabs/rudder-iac/issues/258)) ([3fbfecf](https://github.com/rudderlabs/rudder-iac/commit/3fbfecf4524e9d72f53dc6ea42cf66dfe90f1148))
* **typer:** kotlin generated code supports user provided RudderOptions ([#257](https://github.com/rudderlabs/rudder-iac/issues/257)) ([e1dcb0c](https://github.com/rudderlabs/rudder-iac/commit/e1dcb0c3fbdb4dd9067f70aa5cc79e4d5e6be6f0))
* **typer:** kotlin generation now does not depend on org.jetbrains.kotlin.plugin.serialization for serialization ([#262](https://github.com/rudderlabs/rudder-iac/issues/262)) ([322eb6e](https://github.com/rudderlabs/rudder-iac/commit/322eb6eb31e94b13902c6b7cdea486073c1d1608))
* **typer:** kotlin generator adds ruddertyper context to events ([#202](https://github.com/rudderlabs/rudder-iac/issues/202)) ([61a80b7](https://github.com/rudderlabs/rudder-iac/commit/61a80b712232c5415223afb427db6c771ec3afad))
* **typer:** proper unicode support in Kotlin generated code ([#235](https://github.com/rudderlabs/rudder-iac/issues/235)) ([c52ac1c](https://github.com/rudderlabs/rudder-iac/commit/c52ac1cd29164414f6bedcc009cccd6ea25488bd))
* **typer:** rudder typer adds a disclaimer comment in generated code ([#254](https://github.com/rudderlabs/rudder-iac/issues/254)) ([2fca1f7](https://github.com/rudderlabs/rudder-iac/commit/2fca1f7d225cff0108f99d2db273e8a164df76b1))
* **typer:** rudder typer support for null types ([#251](https://github.com/rudderlabs/rudder-iac/issues/251)) ([75ae5ad](https://github.com/rudderlabs/rudder-iac/commit/75ae5ad3894de74754e5ceb2b7e3b5a1d7e91e93))
* **typer:** rudder typer variants sealed classes ([#205](https://github.com/rudderlabs/rudder-iac/issues/205)) ([d0a21cb](https://github.com/rudderlabs/rudder-iac/commit/d0a21cbbbfd3ec85c45bc023cbe0481da66feb46))
* **typer:** support for 'context.traits' identity section ([#266](https://github.com/rudderlabs/rudder-iac/issues/266)) ([a023bc0](https://github.com/rudderlabs/rudder-iac/commit/a023bc0981c15ae32b13cbe5a276b0610597a6c8))
* **typer:** support for json schema based plan provider ([#193](https://github.com/rudderlabs/rudder-iac/issues/193)) ([655aab8](https://github.com/rudderlabs/rudder-iac/commit/655aab857c38a263f4b6d4c39257b9150336b18b))
* **typer:** support for nested objects in Kotlin generation ([#204](https://github.com/rudderlabs/rudder-iac/issues/204)) ([b79f397](https://github.com/rudderlabs/rudder-iac/commit/b79f397969e77f61e89a86d25fb6b4604499d7ce))
* **typer:** typer command to execute rudder typer bindings generation ([#197](https://github.com/rudderlabs/rudder-iac/issues/197)) ([ad1dbbc](https://github.com/rudderlabs/rudder-iac/commit/ad1dbbc8b9b80e41d0383cad0b97e0ebfda90a9c))
* use new event stream sources APIs ([#188](https://github.com/rudderlabs/rudder-iac/issues/188)) ([86e7c65](https://github.com/rudderlabs/rudder-iac/commit/86e7c65c17c143c564778dc2623a4b116063fd68))


### Bug Fixes

* add missing variant support trackingplan ([#236](https://github.com/rudderlabs/rudder-iac/issues/236)) ([95a8007](https://github.com/rudderlabs/rudder-iac/commit/95a800775f159cbb364e486a826992f7a10c82c1))
* add the custom type as a reference instead of using its name for array of custom types ([#255](https://github.com/rudderlabs/rudder-iac/issues/255)) ([c89e49c](https://github.com/rudderlabs/rudder-iac/commit/c89e49cac7f650e9e436d097b23321a4d5b61872))
* api error to determine if feature is not enabled ([#265](https://github.com/rudderlabs/rudder-iac/issues/265)) ([b64cd05](https://github.com/rudderlabs/rudder-iac/commit/b64cd052bc37a8c2fc67fc9f896f613785e4e791))
* bug while fetching tracking plans with more than 50 pages ([#287](https://github.com/rudderlabs/rudder-iac/issues/287)) ([a4313f5](https://github.com/rudderlabs/rudder-iac/commit/a4313f5efa30bae56e41ceec49c8290e111c82a5))
* building state fails if tracking plan connected to source lacks external id ([#242](https://github.com/rudderlabs/rudder-iac/issues/242)) ([13abbeb](https://github.com/rudderlabs/rudder-iac/commit/13abbeb1ed5e832f3bc21243df06db4ff854a9c9))
* check arrayItemType in addition to property's name and type while validating a project ([#263](https://github.com/rudderlabs/rudder-iac/issues/263)) ([1cbcf91](https://github.com/rudderlabs/rudder-iac/commit/1cbcf9100bb0559f4b6dfe57fb7e828d5857a9c9))
* copy the config before attaching it to state ([#250](https://github.com/rudderlabs/rudder-iac/issues/250)) ([77451b5](https://github.com/rudderlabs/rudder-iac/commit/77451b55b55aab5194d4bee9febc10779151520d))
* enabled field of EventStreamSource is defaulting to false ([#207](https://github.com/rudderlabs/rudder-iac/issues/207)) ([1f84442](https://github.com/rudderlabs/rudder-iac/commit/1f84442fcf7e2ea5130b7bf4d20da28e53990b41))
* fixed an issue during event stream source import when connected to an already managed tracking plan ([#264](https://github.com/rudderlabs/rudder-iac/issues/264)) ([ccad83d](https://github.com/rudderlabs/rudder-iac/commit/ccad83d89babda19d793e917c6e4d4548a06ffaf))
* handle dependencies between CLI managed and non CLI managed resources ([#243](https://github.com/rudderlabs/rudder-iac/issues/243)) ([9e6c85d](https://github.com/rudderlabs/rudder-iac/commit/9e6c85d5b84f85052d0c05fc1fbbe17d4ba58885))
* incorrect field type used when configuring tracking plan connection ([#212](https://github.com/rudderlabs/rudder-iac/issues/212)) ([74cd314](https://github.com/rudderlabs/rudder-iac/commit/74cd314ba4e0c2b0ef75721a15253b37564fbbc9))
* incorrect response schema used in get event stream sources API ([#209](https://github.com/rudderlabs/rudder-iac/issues/209)) ([dfe7aca](https://github.com/rudderlabs/rudder-iac/commit/dfe7aca4d03b908c07629a498c0f861df649575e))
* make property type as optional in validation ([#223](https://github.com/rudderlabs/rudder-iac/issues/223)) ([bfc1013](https://github.com/rudderlabs/rudder-iac/commit/bfc10136429abc475125eae23f1abf2f4b88ac69))
* move the e2e tests to generic apply and destroy ([#211](https://github.com/rudderlabs/rudder-iac/issues/211)) ([6415b95](https://github.com/rudderlabs/rudder-iac/commit/6415b95d49035ccfaa05de6c9fa017cdc60e3d2a))
* ordering for the trackingplan spec test ([#247](https://github.com/rudderlabs/rudder-iac/issues/247)) ([9ce8115](https://github.com/rudderlabs/rudder-iac/commit/9ce81154158487c0af6708879bcb42c11a789fb4))
* panic when casting tracking plan version to float64 ([#210](https://github.com/rudderlabs/rudder-iac/issues/210)) ([7ed1701](https://github.com/rudderlabs/rudder-iac/commit/7ed1701f43101273e7e65ac819d5ffe8bb1deef7))
* process to load state using the stateless cli ([#245](https://github.com/rudderlabs/rudder-iac/issues/245)) ([9474428](https://github.com/rudderlabs/rudder-iac/commit/94744282d047d10b77cd87f3af3b3ee5716eb959))
* reference calculation for the trackingplans and add tests ([#228](https://github.com/rudderlabs/rudder-iac/issues/228)) ([97b0f32](https://github.com/rudderlabs/rudder-iac/commit/97b0f3275507c5f869ea1a2aa65a0cabe8ec7494))
* resolve reference for trackingplan correctly ([#230](https://github.com/rudderlabs/rudder-iac/issues/230)) ([aa984d8](https://github.com/rudderlabs/rudder-iac/commit/aa984d881030438d896a9f3f2b5d8ccc969d6da3))
* set identitySection to properties by default if it is not explicitly set in the project YAMLs ([#217](https://github.com/rudderlabs/rudder-iac/issues/217)) ([62ca1c7](https://github.com/rudderlabs/rudder-iac/commit/62ca1c787db1b27a45b65182bf3d4d30be92c390))
* tests failing in main branch ([#187](https://github.com/rudderlabs/rudder-iac/issues/187)) ([a3dca92](https://github.com/rudderlabs/rudder-iac/commit/a3dca923ed9835f6604a2c491d6e06c419b390eb))
* **typer:** correct serialization of Kotlin enums of non-string types ([#240](https://github.com/rudderlabs/rudder-iac/issues/240)) ([e7a2d1b](https://github.com/rudderlabs/rudder-iac/commit/e7a2d1b7a092562cbe044dc23715874e3bee1181))
* **typer:** fixed bug where custom types only referenced in array item types where not considered during code generation ([#277](https://github.com/rudderlabs/rudder-iac/issues/277)) ([2a22ce3](https://github.com/rudderlabs/rudder-iac/commit/2a22ce384de4f560be734faa076566b5cec52961))
* **typer:** fixed bug with Kotlin property name sanitization and collision handling ([#279](https://github.com/rudderlabs/rudder-iac/issues/279)) ([423ff29](https://github.com/rudderlabs/rudder-iac/commit/423ff29b9bebf2cdbf38188adb8f8c6740c19385))
* **typer:** fixed bug with missing serializers for Lists of Lists ([#288](https://github.com/rudderlabs/rudder-iac/issues/288)) ([347a11f](https://github.com/rudderlabs/rudder-iac/commit/347a11f633005d4effbac1c06e5230f5cd0ba3ea))
* **typer:** fixed issue with custom types nested in custom type arrays ([#283](https://github.com/rudderlabs/rudder-iac/issues/283)) ([1b2b6c8](https://github.com/rudderlabs/rudder-iac/commit/1b2b6c80d3dbe4a55a4134ea6ae69a51703a5e42))
* **typer:** issue with rudderanalytics method name collision ([#268](https://github.com/rudderlabs/rudder-iac/issues/268)) ([9e4dcbf](https://github.com/rudderlabs/rudder-iac/commit/9e4dcbfe701a46e914154ee7b58dd394f7c43a95))
* **typer:** proper escape of comments and strings in Kotlin generation ([#237](https://github.com/rudderlabs/rudder-iac/issues/237)) ([d45d23f](https://github.com/rudderlabs/rudder-iac/commit/d45d23f5b723ba51842d5e1aa5ab99eb58271b2a))
* **typer:** proper handling of $ character in kotlin literals ([#267](https://github.com/rudderlabs/rudder-iac/issues/267)) ([c84786d](https://github.com/rudderlabs/rudder-iac/commit/c84786d59d846f124520d1fc89e14d74d4da7af3))
* **typer:** proper handling of object types to avoid empty data classes ([#238](https://github.com/rudderlabs/rudder-iac/issues/238)) ([6e502b1](https://github.com/rudderlabs/rudder-iac/commit/6e502b1e353fbb504040828d0536ca50584e459c))
* **typer:** rules with unsupported identity sections are skipped instead of generating failed code ([#285](https://github.com/rudderlabs/rudder-iac/issues/285)) ([a0b22c3](https://github.com/rudderlabs/rudder-iac/commit/a0b22c3b18a7031405f31f37300b6d521c6bd693))
* **typer:** use common naming scope for all generated kotlin types ([#231](https://github.com/rudderlabs/rudder-iac/issues/231)) ([1c7ddd1](https://github.com/rudderlabs/rudder-iac/commit/1c7ddd1c22a1d60870f1ee0894d8325bc699d8ab))
* validation fixes for advanced types ([#252](https://github.com/rudderlabs/rudder-iac/issues/252)) ([af27f35](https://github.com/rudderlabs/rudder-iac/commit/af27f3537a07cf3ba89d7cb0a1c01675cc7c526d))


### Miscellaneous

* add support to reconstruct state for custom types and properties ([#185](https://github.com/rudderlabs/rudder-iac/issues/185)) ([a6c191c](https://github.com/rudderlabs/rudder-iac/commit/a6c191c0df0bb9e374675f39bd67c5a0073c7d69))
* apiClient - add projectId for events, categories, properties and custom types ([#156](https://github.com/rudderlabs/rudder-iac/issues/156)) ([c407b52](https://github.com/rudderlabs/rudder-iac/commit/c407b52050dd51a37474b3bfe10270ffc1a3986a))
* apply security best practices by step-security ([#214](https://github.com/rudderlabs/rudder-iac/issues/214)) ([822413b](https://github.com/rudderlabs/rudder-iac/commit/822413b229002c58ec21bc06c54002d3a6cdaa4f))
* apply security best practices from step security ([#271](https://github.com/rudderlabs/rudder-iac/issues/271)) ([5c5e128](https://github.com/rudderlabs/rudder-iac/commit/5c5e1286aa8d4134b00275ae052a247a3c202b96))
* **deps:** bump actions/checkout from 4 to 5 ([#141](https://github.com/rudderlabs/rudder-iac/issues/141)) ([e2a3aaa](https://github.com/rudderlabs/rudder-iac/commit/e2a3aaaf7e6d02aefb9ae59592a7a26c0163a967))
* **deps:** bump actions/setup-go from 5 to 6 ([#163](https://github.com/rudderlabs/rudder-iac/issues/163)) ([ec7d7fc](https://github.com/rudderlabs/rudder-iac/commit/ec7d7fc81bfc4a2b9f4505d46e8ac8f7b78d1710))
* **deps:** bump amannn/action-semantic-pull-request from 5 to 6 ([#143](https://github.com/rudderlabs/rudder-iac/issues/143)) ([1fa71d3](https://github.com/rudderlabs/rudder-iac/commit/1fa71d3455424c591d6d33069a18bde699df90e4))
* **deps:** bump docker/login-action from 3.4.0 to 3.6.0 ([#191](https://github.com/rudderlabs/rudder-iac/issues/191)) ([afecef2](https://github.com/rudderlabs/rudder-iac/commit/afecef226e5778974c8667e8d4b16075eecc16b0))
* **deps:** bump peter-evans/repository-dispatch from 3 to 4 ([#198](https://github.com/rudderlabs/rudder-iac/issues/198)) ([aef9da9](https://github.com/rudderlabs/rudder-iac/commit/aef9da98d52e49acd071c0ac9b0e5abd56d939ac))
* **deps:** bump the go-deps group across 1 directory with 8 updates ([#184](https://github.com/rudderlabs/rudder-iac/issues/184)) ([b5353e5](https://github.com/rudderlabs/rudder-iac/commit/b5353e520b76fae7d137291118c77e361b80d732))
* import apply changes introducing capturing of workspace information and using it in planning operations ([#195](https://github.com/rudderlabs/rudder-iac/issues/195)) ([a7e8e93](https://github.com/rudderlabs/rudder-iac/commit/a7e8e934c4792e43da34592cee9eadc2480f4fb2))
* new workspace importer interface ([#192](https://github.com/rudderlabs/rudder-iac/issues/192)) ([6263dcc](https://github.com/rudderlabs/rudder-iac/commit/6263dcc24b8cc1c51a60fcf3078fcc362cdd578d))
* reconstruct state for tracking plans ([#194](https://github.com/rudderlabs/rudder-iac/issues/194)) ([f8c64e5](https://github.com/rudderlabs/rudder-iac/commit/f8c64e535b95474dea5733363b33e984917af051))
* refactor lister with Options pattern ([#246](https://github.com/rudderlabs/rudder-iac/issues/246)) ([0b019ec](https://github.com/rudderlabs/rudder-iac/commit/0b019ece340282c723febad56af54bf309173e1b))
* syncer no longer updates/delete state during applies ([#284](https://github.com/rudderlabs/rudder-iac/issues/284)) ([be200ea](https://github.com/rudderlabs/rudder-iac/commit/be200ea81f42ec58fbaad3889b845f146200a993))
* **typer:** kotlin generated code now uses interfaces instead of abstract classes for serializers ([#259](https://github.com/rudderlabs/rudder-iac/issues/259)) ([f1400f3](https://github.com/rudderlabs/rudder-iac/commit/f1400f320999577312afd4248f3fa0ddc9de8098))
* **typer:** remove typer experimental flag ([#281](https://github.com/rudderlabs/rudder-iac/issues/281)) ([2c68ef9](https://github.com/rudderlabs/rudder-iac/commit/2c68ef94eccca10d231ac5ddf3d4f63e03f7750e))

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
