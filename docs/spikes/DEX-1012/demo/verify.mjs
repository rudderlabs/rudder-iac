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
const rootSchema = JSON.parse(
  fs.readFileSync(path.join(schemaDir, "rudder-spec.schema.json"), "utf8"),
);

assert(rootSchema.oneOf.length >= 3, "root fixture must exercise kind discrimination");

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

async function hover(relativePath, needle, schemas = []) {
  const sample = loadSample(relativePath);
  const doc = document(sample.filePath, sample.contents);
  const offset = sample.contents.lastIndexOf(needle);
  assert.notEqual(offset, -1, `hover target ${needle} must exist`);
  const result = await service(schemas).doHover(doc, doc.positionAt(offset + 1));
  return typeof result?.contents === "string"
    ? result.contents
    : JSON.stringify(result?.contents ?? "");
}

async function diagnostics(relativePath, transform = (contents) => contents, schemas = []) {
  const sample = loadSample(relativePath);
  const contents = transform(sample.contents);
  return service(schemas).doValidation(document(sample.filePath, contents), false);
}

function settingsAssociation() {
  return [{
    uri: pathToFileURL(path.join(schemaDir, "event-stream-source.schema.json")).href,
    fileMatch: ["settings-associated.yaml"],
  }];
}

function rootAssociation() {
  return [{
    uri: pathToFileURL(path.join(schemaDir, "rudder-spec.schema.json")).href,
    fileMatch: ["samples/root-comparison/*.yaml"],
  }];
}

const matrix = [];
const topLevelTransform = (contents) => contents.slice(0, contents.indexOf("version:"));
const beforeKindTransform = (contents) => contents.slice(0, contents.indexOf("kind:"));

for (const sample of ["sources/mobile.yaml"]) {
  assert.deepEqual(await diagnostics(sample), [], `${sample} must validate against its modeline schema`);
}
for (const sample of ["root-comparison/mobile.yaml", "root-comparison/customer-360.yaml"]) {
  assert.deepEqual(await diagnostics(sample, (contents) => contents, rootAssociation()), [], `${sample} must validate through the root mapping`);
}
assert.deepEqual(
  await diagnostics("settings-associated.yaml", (contents) => contents, settingsAssociation()),
  [],
  "settings-associated.yaml must validate through yaml.schemas",
);
matrix.push({ sample: "all", association: "modelines and yaml.schemas equivalent", case: "fixture validation", expected: "no diagnostics", result: "pass" });

for (const [sample, schemas] of [["root-comparison/mobile.yaml", rootAssociation()], ["sources/mobile.yaml", []]]) {
  const blank = await completionLabels(sample, topLevelTransform, schemas);
  assert(blank.includes("version") && blank.includes("kind") && blank.includes("spec"));
  matrix.push({ sample, association: schemas.length ? "root yaml.schemas equivalent" : "per-kind modeline", case: "blank top level", expected: "version, kind, metadata, spec", result: "pass" });
}

for (const [sample, schemas] of [["root-comparison/mobile.yaml", rootAssociation()], ["sources/mobile.yaml", []]]) {
  const beforeKind = await completionLabels(sample, beforeKindTransform, schemas);
  assert(beforeKind.includes("kind") && beforeKind.includes("metadata") && beforeKind.includes("spec"));
  matrix.push({ sample, association: schemas.length ? "root yaml.schemas equivalent" : "per-kind modeline", case: "top level before kind", expected: "kind, metadata, spec", result: "pass" });
}

const sourceSpecTransform = (contents) => contents.slice(0, contents.indexOf("  id:"));
const sourceRoot = await completionLabels("root-comparison/mobile.yaml", sourceSpecTransform, rootAssociation());
assert(sourceRoot.includes("enabled") && sourceRoot.includes("name") && sourceRoot.includes("type"));
assert(!sourceRoot.includes("account_id") && !sourceRoot.includes("models"));
matrix.push({ sample: "root-comparison/mobile.yaml", association: "root yaml.schemas equivalent", case: "source spec completion", expected: "includes source fields; excludes data-graph fields", result: "pass" });

const graphRoot = await completionLabels("root-comparison/customer-360.yaml", sourceSpecTransform, rootAssociation());
assert(graphRoot.includes("account_id") && graphRoot.includes("models"));
assert(!graphRoot.includes("enabled") && !graphRoot.includes("name"));
matrix.push({ sample: "root-comparison/customer-360.yaml", association: "root yaml.schemas equivalent", case: "data-graph spec completion", expected: "includes data-graph fields; excludes source fields", result: "pass" });

const sourcePerKind = await completionLabels("sources/mobile.yaml", sourceSpecTransform);
assert(sourcePerKind.includes("enabled") && sourcePerKind.includes("name") && sourcePerKind.includes("type"));
matrix.push({ sample: "sources/mobile.yaml", association: "per-kind modeline", case: "source spec completion", expected: "source fields", result: "pass" });

const settingsCompletion = await completionLabels("settings-associated.yaml", sourceSpecTransform, settingsAssociation());
assert(settingsCompletion.includes("enabled") && settingsCompletion.includes("name"));
matrix.push({ sample: "settings-associated.yaml", association: "yaml.schemas equivalent", case: "source spec completion", expected: "source fields", result: "pass" });

const changedKind = await diagnostics("root-comparison/mobile.yaml", (contents) => contents.replace("kind: event-stream-source", "kind: data-graph"), rootAssociation());
assert(changedKind.some((item) => item.message.includes("Property name is not allowed")));
assert(changedKind.some((item) => item.message.includes("Missing property \"account_id\"")));
matrix.push({ sample: "root-comparison/mobile.yaml", association: "root yaml.schemas equivalent", case: "kind changed with source fields", expected: "wrong-kind and missing-field diagnostics", result: "pass" });

const missingKind = await diagnostics("root-comparison/mobile.yaml", (contents) => contents.replace("kind: event-stream-source\n", ""), rootAssociation());
assert(missingKind.some((item) => item.message.includes("Missing property \"kind\"")));
const missingKindCompletions = await completionLabels(
  "root-comparison/mobile.yaml",
  (contents) => sourceSpecTransform(contents.replace("kind: event-stream-source\n", "")),
  rootAssociation(),
);
assert(missingKindCompletions.includes("enabled") && missingKindCompletions.includes("account_id"));
matrix.push({ sample: "root-comparison/mobile.yaml", association: "root yaml.schemas equivalent", case: "spec completion before kind", expected: "mixed suggestions from multiple kinds; missing-kind diagnostic", result: "pass" });

const sourceHover = await hover("root-comparison/mobile.yaml", "enabled", rootAssociation());
assert(sourceHover.includes("RudderStack event-stream-source spec"));
matrix.push({ sample: "root-comparison/mobile.yaml", association: "root yaml.schemas equivalent", case: "hover on spec.enabled", expected: "selected schema title and source link", result: "pass" });

const perKindHover = await hover("sources/mobile.yaml", "enabled");
assert(perKindHover.includes("RudderStack event-stream-source spec"));
matrix.push({ sample: "sources/mobile.yaml", association: "per-kind modeline", case: "hover on spec.enabled", expected: "schema title and source link", result: "pass" });

console.log(JSON.stringify({
  runtime: {
    schemaGeneratorCommit: "bbcee89",
    rootSchemaBranches: rootSchema.oneOf.length,
    yamlLanguageServer: yamlLanguageServerPackage.version,
    vscodeLanguageserverTextdocument: textDocumentPackage.version,
    node: process.version,
  },
  matrix,
}, null, 2));
