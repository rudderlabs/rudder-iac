"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode_1 = require("vscode");
const client_1 = require("./client");
const config_1 = require("./config");
let client;
/**
 * Activates the extension
 */
async function activate(context) {
    console.log('Activating Rudder YAML extension');
    // Check if LSP is enabled
    if (!(0, config_1.isEnabled)()) {
        console.log('Rudder LSP is disabled in configuration');
        return;
    }
    // Check if rudder-cli is available
    const serverPath = (0, config_1.getServerPath)();
    if (!(await isRudderCliAvailable(serverPath))) {
        const message = `Rudder CLI not found at "${serverPath}". Please install Rudder CLI or update the "rudder.lsp.serverPath" setting.`;
        vscode_1.window.showErrorMessage(message);
        return;
    }
    try {
        // Create and start the language client
        client = (0, client_1.createLanguageClient)(context);
        await (0, client_1.startClient)();
        // Register commands
        registerCommands(context);
        // Watch for configuration changes
        watchConfiguration(context);
        console.log('Rudder YAML extension activated successfully');
    }
    catch (error) {
        console.error('Failed to activate Rudder YAML extension:', error);
        vscode_1.window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
    }
}
/**
 * Deactivates the extension
 */
async function deactivate() {
    console.log('Deactivating Rudder YAML extension');
    if (client) {
        await (0, client_1.stopClient)();
        client = undefined;
    }
}
/**
 * Registers extension commands
 */
function registerCommands(context) {
    // Restart LSP server command
    context.subscriptions.push(vscode_1.commands.registerCommand('rudder.restartServer', async () => {
        if (client) {
            try {
                await (0, client_1.restartClient)(context);
                vscode_1.window.showInformationMessage('Rudder LSP server restarted successfully');
            }
            catch (error) {
                vscode_1.window.showErrorMessage(`Failed to restart Rudder LSP server: ${error}`);
            }
        }
    }));
    // Show server status command
    context.subscriptions.push(vscode_1.commands.registerCommand('rudder.showStatus', () => {
        const status = (0, client_1.isClientRunning)() ? 'Running' : 'Stopped';
        vscode_1.window.showInformationMessage(`Rudder LSP Server Status: ${status}`);
    }));
}
/**
 * Watches for configuration changes and restarts the server if needed
 */
function watchConfiguration(context) {
    context.subscriptions.push(vscode_1.workspace.onDidChangeConfiguration(async (event) => {
        if (event.affectsConfiguration('rudder.lsp')) {
            const enabled = (0, config_1.isEnabled)();
            if (!enabled && (0, client_1.isClientRunning)()) {
                // LSP was disabled
                await (0, client_1.stopClient)();
                vscode_1.window.showInformationMessage('Rudder LSP server stopped (disabled in settings)');
            }
            else if (enabled && !(0, client_1.isClientRunning)()) {
                // LSP was enabled
                try {
                    client = (0, client_1.createLanguageClient)(context);
                    await (0, client_1.startClient)();
                    vscode_1.window.showInformationMessage('Rudder LSP server started');
                }
                catch (error) {
                    vscode_1.window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
                }
            }
            else if (enabled && (0, client_1.isClientRunning)()) {
                // Settings changed while running - restart to pick up new settings
                const newServerPath = (0, config_1.getServerPath)();
                const currentServerPath = (0, config_1.getServerPath)(); // This might need adjustment
                if (newServerPath !== currentServerPath) {
                    try {
                        await (0, client_1.restartClient)(context);
                        vscode_1.window.showInformationMessage('Rudder LSP server restarted with new settings');
                    }
                    catch (error) {
                        vscode_1.window.showErrorMessage(`Failed to restart Rudder LSP server: ${error}`);
                    }
                }
            }
        }
    }));
}
/**
 * Checks if rudder-cli binary is available
 */
async function isRudderCliAvailable(serverPath) {
    const { spawn } = require('child_process');
    return new Promise((resolve) => {
        const process = spawn(serverPath, ['--version'], {
            stdio: 'ignore',
            timeout: 5000
        });
        process.on('close', (code) => {
            resolve(code === 0);
        });
        process.on('error', () => {
            resolve(false);
        });
    });
}
//# sourceMappingURL=extension.js.map