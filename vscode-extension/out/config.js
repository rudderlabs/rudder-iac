"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getServerPath = getServerPath;
exports.isEnabled = isEnabled;
exports.getServerCommand = getServerCommand;
const vscode_1 = require("vscode");
/**
 * Gets the server path from workspace configuration
 */
function getServerPath() {
    const config = vscode_1.workspace.getConfiguration('rudder.lsp');
    return config.get('serverPath', 'rudder-cli');
}
/**
 * Checks if LSP is enabled from workspace configuration
 */
function isEnabled() {
    const config = vscode_1.workspace.getConfiguration('rudder.lsp');
    return config.get('enabled', true);
}
/**
 * Gets the full server command array
 */
function getServerCommand() {
    const serverPath = getServerPath();
    return [serverPath, 'lsp'];
}
//# sourceMappingURL=config.js.map