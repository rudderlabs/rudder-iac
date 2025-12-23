import { ExtensionContext, workspace, window, commands } from 'vscode';
import { createLanguageClient, startClient, stopClient, restartClient, isClientRunning } from './client';
import { getServerPath, isEnabled } from './config';

let client: any;

/**
 * Activates the extension
 */
export async function activate(context: ExtensionContext): Promise<void> {
    console.log('Activating Rudder YAML extension');

    // Check if LSP is enabled
    if (!isEnabled()) {
        console.log('Rudder LSP is disabled in configuration');
        return;
    }

    // Check if rudder-cli is available
    const serverPath = getServerPath();
    if (!(await isRudderCliAvailable(serverPath))) {
        const message = `Rudder CLI not found at "${serverPath}". Please install Rudder CLI or update the "rudder.lsp.serverPath" setting.`;
        window.showErrorMessage(message);
        return;
    }

    try {
        // Create and start the language client
        client = createLanguageClient(context);
        await startClient();

        // Register commands
        registerCommands(context);

        // Watch for configuration changes
        watchConfiguration(context);

        console.log('Rudder YAML extension activated successfully');
    } catch (error) {
        console.error('Failed to activate Rudder YAML extension:', error);
        window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
    }
}

/**
 * Deactivates the extension
 */
export async function deactivate(): Promise<void> {
    console.log('Deactivating Rudder YAML extension');

    if (client) {
        await stopClient();
        client = undefined;
    }
}

/**
 * Registers extension commands
 */
function registerCommands(context: ExtensionContext): void {
    // Restart LSP server command
    context.subscriptions.push(
        commands.registerCommand('rudder.restartServer', async () => {
            if (client) {
                try {
                    await restartClient(context);
                    window.showInformationMessage('Rudder LSP server restarted successfully');
                } catch (error) {
                    window.showErrorMessage(`Failed to restart Rudder LSP server: ${error}`);
                }
            }
        })
    );

    // Show server status command
    context.subscriptions.push(
        commands.registerCommand('rudder.showStatus', () => {
            const status = isClientRunning() ? 'Running' : 'Stopped';
            window.showInformationMessage(`Rudder LSP Server Status: ${status}`);
        })
    );
}

/**
 * Watches for configuration changes and restarts the server if needed
 */
function watchConfiguration(context: ExtensionContext): void {
    context.subscriptions.push(
        workspace.onDidChangeConfiguration(async (event) => {
            if (event.affectsConfiguration('rudder.lsp')) {
                const enabled = isEnabled();

                if (!enabled && isClientRunning()) {
                    // LSP was disabled
                    await stopClient();
                    window.showInformationMessage('Rudder LSP server stopped (disabled in settings)');
                } else if (enabled && !isClientRunning()) {
                    // LSP was enabled
                    try {
                        client = createLanguageClient(context);
                        await startClient();
                        window.showInformationMessage('Rudder LSP server started');
                    } catch (error) {
                        window.showErrorMessage(`Failed to start Rudder LSP server: ${error}`);
                    }
                } else if (enabled && isClientRunning()) {
                    // Settings changed while running - restart to pick up new settings
                    const newServerPath = getServerPath();
                    const currentServerPath = getServerPath(); // This might need adjustment

                    if (newServerPath !== currentServerPath) {
                        try {
                            await restartClient(context);
                            window.showInformationMessage('Rudder LSP server restarted with new settings');
                        } catch (error) {
                            window.showErrorMessage(`Failed to restart Rudder LSP server: ${error}`);
                        }
                    }
                }
            }
        })
    );
}

/**
 * Checks if rudder-cli binary is available
 */
async function isRudderCliAvailable(serverPath: string): Promise<boolean> {
    const { spawn } = require('child_process');

    return new Promise((resolve) => {
        const process = spawn(serverPath, ['--version'], {
            stdio: 'ignore',
            timeout: 5000
        });

        process.on('close', (code: number) => {
            resolve(code === 0);
        });

        process.on('error', () => {
            resolve(false);
        });
    });
}
