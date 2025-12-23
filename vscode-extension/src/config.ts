import { workspace } from 'vscode';

/**
 * Gets the server path from workspace configuration
 */
export function getServerPath(): string {
    const config = workspace.getConfiguration('rudder.lsp');
    return config.get('serverPath', 'rudder-cli');
}

/**
 * Checks if LSP is enabled from workspace configuration
 */
export function isEnabled(): boolean {
    const config = workspace.getConfiguration('rudder.lsp');
    return config.get('enabled', true);
}

/**
 * Gets the full server command array
 */
export function getServerCommand(): string[] {
    const serverPath = getServerPath();
    return [serverPath, 'lsp'];
}
