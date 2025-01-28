# rudder-cli

## Table of Contents

- [⚠️ Work in Progress](#️-work-in-progress)
- [Installation](#installation)
  - [MacOS](#macos)
    - [Apple Silicon](#apple-silicon)
    - [Intel-based](#intel-based)
  - [Linux](#linux)
  - [Build from Source](#build-from-source)

## ⚠️ Work in Progress

> **Warning**
>
> Please note that this tool is currently a work in progress. We are actively developing and improving it, and as such, there may be frequent changes and updates. We do not guarantee backwards compatibility at this stage.

## Installation

### MacOS

#### Apple Silicon

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/download/v0.2.0/rudder-cli_Darwin_arm64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

#### Intel-based

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/download/v0.2.0/rudder-cli_Darwin_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Linux

```sh
curl -L https://github.com/rudderlabs/rudder-iac/releases/download/v0.2.0/rudder-cli_Linux_x86_64.tar.gz | tar -xz rudder-cli
sudo mv rudder-cli /usr/local/bin/
```

### Build from Source

To build the `rudder-cli` from source, you need to have Go installed. Then, run the following commands:

```sh
git clone https://github.com/rudderlabs/rudder-iac.git
cd rudder-iac
make build
sudo mv bin/rudder-cli /usr/local/bin/
```
