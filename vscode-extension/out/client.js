"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.createLanguageClient = createLanguageClient;
exports.startClient = startClient;
exports.stopClient = stopClient;
exports.restartClient = restartClient;
exports.isClientRunning = isClientRunning;
const vscode_1 = require("vscode");
const node_1 = require("vscode-languageclient/node");
const path = __importStar(require("path"));
let client;
let statusBarItem;
/**
 * Creates and configures a LanguageClient for the Rudder LSP server
 */
function createLanguageClient(context) {
    // Server options - spawn rudder-cli lsp via stdio
    const serverOptions = {
        command: path.join(context.extensionPath, 'bin', 'rudder-cli'),
        args: ['lsp'],
        options: {
            shell: true
        }
    };
    // Client options - configure for YAML files
    const clientOptions = {
        // Register the server for YAML documents
        documentSelector: [
            { scheme: 'file', language: 'yaml' },
            { scheme: 'file', pattern: '**/*.yml' }
        ],
        outputChannelName: 'Rudder LSP'
    };
    // Create the language client
    client = new node_1.LanguageClient('rudderLSP', 'Rudder YAML Language Server', serverOptions, clientOptions);
    // Set up status bar
    setupStatusBar(context);
    return client;
}
/**
 * Starts the language client
 */
async function startClient() {
    if (!client) {
        throw new Error('Language client not initialized');
    }
    try {
        updateStatusBar('Starting...', 'warning');
        await client.start();
        updateStatusBar('Running', 'success');
    }
    catch (error) {
        updateStatusBar('Failed', 'error');
        vscode_1.window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
        throw error;
    }
}
/**
 * Stops the language client
 */
async function stopClient() {
    if (client) {
        try {
            updateStatusBar('Stopping...', 'warning');
            await client.stop();
            updateStatusBar('Stopped', 'inactive');
        }
        catch (error) {
            updateStatusBar('Error', 'error');
            vscode_1.window.showErrorMessage(`Failed to stop Rudder LSP server: ${error}`);
        }
    }
}
/**
 * Restarts the language client
 */
async function restartClient(context) {
    await stopClient();
    client = createLanguageClient(context);
    await startClient();
}
/**
 * Sets up the status bar item
 */
function setupStatusBar(context) {
    statusBarItem = vscode_1.window.createStatusBarItem(vscode_1.StatusBarAlignment.Right, 100);
    statusBarItem.command = 'rudder.restartLsp';
    context.subscriptions.push(statusBarItem);
    // Register restart command
    context.subscriptions.push(vscode_1.commands.registerCommand('rudder.restartLsp', async () => {
        if (context) {
            await restartClient(context);
            vscode_1.window.showInformationMessage('Rudder LSP server restarted');
        }
    }));
    updateStatusBar('Inactive', 'inactive');
    statusBarItem.show();
}
/**
 * Updates the status bar with the given text and color
 */
function updateStatusBar(text, state) {
    if (!statusBarItem)
        return;
    statusBarItem.text = `$(server) Rudder LSP: ${text}`;
    switch (state) {
        case 'success':
            statusBarItem.backgroundColor = new vscode_1.ThemeColor('statusBarItem.activeBackground');
            break;
        case 'warning':
            statusBarItem.backgroundColor = new vscode_1.ThemeColor('statusBarItem.warningBackground');
            break;
        case 'error':
            statusBarItem.backgroundColor = new vscode_1.ThemeColor('statusBarItem.errorBackground');
            break;
        case 'inactive':
        default:
            statusBarItem.backgroundColor = undefined;
            break;
    }
}
/**
 * Checks if the language client is running
 */
function isClientRunning() {
    return client !== undefined && client.state === 2; // Running state
}
//# sourceMappingURL=client.js.map