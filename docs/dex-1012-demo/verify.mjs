#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { createRequire } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";

const dependencyRoot = process.env.DEX_1012_YLS_ROOT ?? "/tmp/dex-1012-yls";
const require = createRequire(path.join(dependencyRoot, "package.json"));
const { getLanguageService } = require("yaml-language-server/out/server/src/languageservice/yamlLanguageService.js");
const { yamlDocumentsCache } = require("yaml-language-server/out/server/src/languageservice/parser/yaml-documents.js");
const { TextDocument } = require("vscode-languageserver-textdocument");
const yamlLanguageServerPackage = JSON.parse(
  fs.readFileSync(path.join(dependencyRoot, "node_modules/yaml-language-server/package.json"), "utf8"),
);
const textDocumentPackage = JSON.parse(
  fs.readFileSync(path.join(dependencyRoot, "node_modules/vscode-languageserver-textdocument/package.json"), "utf8"),
);

const demoDir = path.dirname(fileURLToPath(import.meta.url));
const samplesDir = path.join(demoDir, "samples");
const schemaDir = path.join(demoDir, "schemas");

function schemaRequestService(uri) {
  return fs.promises.readFile(fileURLToPath(uri), "utf8");
}

function service(schemas = []) {
  const languageService = getLanguageService({
    schemaRequestService,
    clientCapabilities: {},
    yamlSettings: {},
  });
  languageService.configure({
    validate: true,
    hover: true,
    completion: true,
    customTags: [],
    yamlVersion: "1.2",
    schemas,
  });
  return languageService;
}

function document(filePath, contents) {
  yamlDocumentsCache.clear();
  return TextDocument.create(pathToFileURL(filePath).href, "yaml", 1, contents);
}

function loadSample(relativePath) {
  const filePath = path.join(samplesDir, relativePath);
  return {
    filePath,
    contents: fs.readFileSync(filePath, "utf8"),
  };
}

async function completionLabels(relativePath, transform = (contents) => contents, schemas = []) {
  const sample = loadSample(relativePath);
  const contents = transform(sample.contents);
  const doc = document(sample.filePath, contents);
  const result = await service(schemas).doComplete(
    doc,
    doc.positionAt(contents.length),
    false,
  );
  return [...new Set((result?.items ?? []).map((item) => item.label))].sort();
}

async function hover(relativePath, needle) {
  const sample = loadSample(relativePath);
  const doc = document(sample.filePath, sample.contents);
  const offset = sample.contents.lastIndexOf(needle);
  assert.notEqual(offset, -1, `hover target ${needle} must exist`);
  const result = await service().doHover(doc, doc.positionAt(offset + 1));
  return typeof result?.contents === "string"
    ? result.contents
    : JSON.stringify(result?.contents ?? "");
}

async function diagnostics(relativePath, transform = (contents) => contents) {
  const sample = loadSample(relativePath);
  const contents = transform(sample.contents);
  return service().doValidation(document(sample.filePath, contents), false);
}

function settingsAssociation() {
  return [{
    uri: pathToFileURL(path.join(schemaDir, "event-stream-source.json")).href,
    fileMatch: ["settings-associated.yaml"],
  }];
}

const matrix = [];
const topLevelTransform = (contents) => contents.slice(0, contents.indexOf("version:"));

for (const sample of ["sources/mobile.yaml", "per-kind-source.yaml"]) {
  const blank = await completionLabels(sample, topLevelTransform);
  assert(blank.includes("version") && blank.includes("kind") && blank.includes("spec"));
  matrix.push({ sample, association: "modeline", case: "blank top level", expected: "version, kind, metadata, spec", result: "pass" });
}

const sourceSpecTransform = (contents) => contents.slice(0, contents.indexOf("  id:"));
const sourceRoot = await completionLabels("sources/mobile.yaml", sourceSpecTransform);
assert(sourceRoot.includes("enabled") && sourceRoot.includes("name") && sourceRoot.includes("type"));
assert(!sourceRoot.includes("account_id") && !sourceRoot.includes("models"));
matrix.push({ sample: "sources/mobile.yaml", association: "root modeline", case: "source spec completion", expected: "includes source fields; excludes data-graph fields", result: "pass" });

const graphRoot = await completionLabels("customer-360.yaml", sourceSpecTransform);
assert(graphRoot.includes("account_id") && graphRoot.includes("models"));
assert(!graphRoot.includes("enabled") && !graphRoot.includes("name"));
matrix.push({ sample: "customer-360.yaml", association: "root modeline", case: "data-graph spec completion", expected: "includes data-graph fields; excludes source fields", result: "pass" });

const sourcePerKind = await completionLabels("per-kind-source.yaml", sourceSpecTransform);
assert(sourcePerKind.includes("enabled") && sourcePerKind.includes("name") && sourcePerKind.includes("type"));
matrix.push({ sample: "per-kind-source.yaml", association: "per-kind modeline", case: "source spec completion", expected: "source fields", result: "pass" });

const settingsCompletion = await completionLabels("settings-associated.yaml", sourceSpecTransform, settingsAssociation());
assert(settingsCompletion.includes("enabled") && settingsCompletion.includes("name"));
matrix.push({ sample: "settings-associated.yaml", association: "yaml.schemas equivalent", case: "source spec completion", expected: "source fields", result: "pass" });

const changedKind = await diagnostics("sources/mobile.yaml", (contents) => contents.replace("kind: event-stream-source", "kind: data-graph"));
assert(changedKind.some((item) => item.message.includes("Property name is not allowed")));
assert(changedKind.some((item) => item.message.includes("Missing property \"account_id\"")));
matrix.push({ sample: "sources/mobile.yaml", association: "root modeline", case: "kind changed with source fields", expected: "wrong-kind and missing-field diagnostics", result: "pass" });

const missingKind = await diagnostics("sources/mobile.yaml", (contents) => contents.replace("kind: event-stream-source\n", ""));
assert(missingKind.some((item) => item.message.includes("Missing property \"kind\"")));
const missingKindCompletions = await completionLabels(
  "sources/mobile.yaml",
  (contents) => sourceSpecTransform(contents.replace("kind: event-stream-source\n", "")),
);
assert(!missingKindCompletions.includes("enabled") && !missingKindCompletions.includes("account_id"));
matrix.push({ sample: "sources/mobile.yaml", association: "root modeline", case: "kind missing", expected: "no kind-specific suggestions; missing-kind diagnostic", result: "pass" });

const sourceHover = await hover("sources/mobile.yaml", "enabled");
assert(sourceHover.includes("Whether this source accepts events"));
matrix.push({ sample: "sources/mobile.yaml", association: "root modeline", case: "hover on spec.enabled", expected: "field description", result: "pass" });

const perKindHover = await hover("per-kind-source.yaml", "enabled");
assert(perKindHover.includes("Whether this source accepts events"));
matrix.push({ sample: "per-kind-source.yaml", association: "per-kind modeline", case: "hover on spec.enabled", expected: "field description", result: "pass" });

console.log(JSON.stringify({
  runtime: {
    yamlLanguageServer: yamlLanguageServerPackage.version,
    vscodeLanguageserverTextdocument: textDocumentPackage.version,
    node: process.version,
  },
  matrix,
}, null, 2));
