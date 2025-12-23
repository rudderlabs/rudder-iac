import {
    workspace,
    ExtensionContext,
    window,
    commands,
    StatusBarAlignment,
    StatusBarItem,
    ThemeColor
} from 'vscode';
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind,
    Executable
} from 'vscode-languageclient/node';
import * as path from 'path';

import { getServerCommand, isEnabled } from './config';

let client: LanguageClient | undefined;
let statusBarItem: StatusBarItem;

/**
 * Creates and configures a LanguageClient for the Rudder LSP server
 */
export function createLanguageClient(context: ExtensionContext): LanguageClient {
    // Server options - spawn rudder-cli lsp via stdio
    const serverOptions: ServerOptions = {
        command: path.join(context.extensionPath, 'bin', 'rudder-cli'),
        args: ['lsp'],
        options: {
            shell: true
        }
    };

    // Client options - configure for YAML files
    const clientOptions: LanguageClientOptions = {
        // Register the server for YAML documents
        documentSelector: [
            { scheme: 'file', language: 'yaml' },
            { scheme: 'file', pattern: '**/*.yml' }
        ],
        synchronize: {
            // Notify the server about file changes to '.yaml' and '.yml' files
            fileEvents: workspace.createFileSystemWatcher('**/*.{yaml,yml}')
        },
        outputChannelName: 'Rudder LSP'
    };

    // Create the language client
    client = new LanguageClient(
        'rudderLSP',
        'Rudder YAML Language Server',
        serverOptions,
        clientOptions
    );

    // Set up status bar
    setupStatusBar(context);

    return client;
}

/**
 * Starts the language client
 */
export async function startClient(): Promise<void> {
    if (!client) {
        throw new Error('Language client not initialized');
    }

    try {
        updateStatusBar('Starting...', 'warning');
        await client.start();
        updateStatusBar('Running', 'success');
    } catch (error) {
        updateStatusBar('Failed', 'error');
        window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
        throw error;
    }
}

/**
 * Stops the language client
 */
export async function stopClient(): Promise<void> {
    if (client) {
        try {
            updateStatusBar('Stopping...', 'warning');
            await client.stop();
            updateStatusBar('Stopped', 'inactive');
        } catch (error) {
            updateStatusBar('Error', 'error');
            window.showErrorMessage(`Failed to stop Rudder LSP server: ${error}`);
        }
    }
}

/**
 * Restarts the language client
 */
export async function restartClient(context: ExtensionContext): Promise<void> {
    await stopClient();
    client = createLanguageClient(context);
    await startClient();
}

/**
 * Sets up the status bar item
 */
function setupStatusBar(context: ExtensionContext): void {
    statusBarItem = window.createStatusBarItem(StatusBarAlignment.Right, 100);
    statusBarItem.command = 'rudder.restartLsp';
    context.subscriptions.push(statusBarItem);

    // Register restart command
    context.subscriptions.push(
        commands.registerCommand('rudder.restartLsp', async () => {
            if (context) {
                await restartClient(context);
                window.showInformationMessage('Rudder LSP server restarted');
            }
        })
    );

    updateStatusBar('Inactive', 'inactive');
    statusBarItem.show();
}

/**
 * Updates the status bar with the given text and color
 */
function updateStatusBar(text: string, state: 'success' | 'warning' | 'error' | 'inactive'): void {
    if (!statusBarItem) return;

    statusBarItem.text = `$(server) Rudder LSP: ${text}`;

    switch (state) {
        case 'success':
            statusBarItem.backgroundColor = new ThemeColor('statusBarItem.activeBackground');
            break;
        case 'warning':
            statusBarItem.backgroundColor = new ThemeColor('statusBarItem.warningBackground');
            break;
        case 'error':
            statusBarItem.backgroundColor = new ThemeColor('statusBarItem.errorBackground');
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
export function isClientRunning(): boolean {
    return client !== undefined && client.state === 2; // Running state
}
