---
title: Precompiles
description: Precompile are MEVM contracts that are implemented in native code instead of bytecode.
custom_edit_url: "https://github.com/flashbots/suave-specs/edit/main/specs/rigil/precompiles.md"
---

<div className="hide-in-docs">

<!-- omit from toc -->
# Precompiles

<!-- TOC -->

- [Overview](#overview)
- [Available Precompiles](#available-precompiles)
  - [`IsConfidential`](#isconfidential){{range .Functions}}
  - [`{{.Name}}`](#{{.Name}}){{end}}
- [Precompiles Governance](#precompiles-governance)

<!-- /TOC -->

---

## Overview

</div>

Precompile are MEVM contracts that are implemented in native code instead of bytecode. Precompiles additionally can communicate with internal APIs. Currently the MEVM supports all existing Ethereum Precompiles up to Dencun, and introduces four new classes of precompiles:

1. offchain computation that is too expensive in solidity
2. calls to API methods to interact with the Confidential Data Store
3. calls to `suavex` API Methods to interact with Domain-Specific Services
4. calls to retrieve context for the confidential compute requests

## Available Precompiles

A list of available precompiles in Rigil are as follows:

### `IsConfidential`

Address: `0x0000000000000000000000000000000042010000`

Determines if the current execution mode is regular (onchain) or confidential. Outputs a boolean value.

```solidity
function isConfidential() internal view returns (bool b)
```

{{range .Functions}}
### `{{.Name}}`

Address: `{{.Address}}`

{{.Description}}

```solidity
function {{.Name}}({{range .Input}}{{styp .Typ}} {{.Name}}, {{end}}) internal view returns ({{range .Output.Fields}}{{styp .Typ}}, {{end}})
```
{{end}}

## Precompiles Governance

The governance process for adding precompiles is in it's early stages but is as follows:
- Discuss the idea in a [forum post](https://collective.flashbots.net/)
- Open a PR and provide implementation
- Feedback and review
- Possibly merge and deploy in the next network upgrade, or sooner, depending on the precompile
