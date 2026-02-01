/**
 * ChartDB Sync Toolbar - Independent Implementation for SYNC operations
 * 
 * This file contains duplicated logic from ChartDB's export-import-utils.ts
 * to ensure independence from ChartDB internals (which is a submodule/external dependency).
 * 
 * IMPORTANT DIFFERENCE FROM ChartDB EXPORT:
 * - ChartDB's export creates NEW IDs for all entities (for file export/import)
 * - This sync version PRESERVES the diagram ID to avoid duplicates during sync
 * 
 * Copied functions:
 * - runningIdGenerator() -> createRunningIdGenerator()
 * - cloneDiagram() -> cloneDiagramForExport() [MODIFIED: preserves diagram ID]
 * - diagramToJSONOutput() -> diagramToJSON() [MODIFIED: preserves diagram ID]
 * 
 * Source: chartdb/src/lib/export-import-utils.ts
 */

(function() {
    'use strict';

    const CONFIG = {
        apiBaseUrl: '/sync/api',
        debounceDelay: 2000,
        storageKeys: {
            autoSync: 'chartdb_sync_auto'
        }
    };

    let state = {
        isAuthenticated: false,
        autoSyncEnabled: true,
        syncStatus: 'idle',
        lastSyncTime: null,
        currentDiagramId: null,
        currentDiagramName: null,
        debounceTimer: null,
    };

    // API Client
    class SyncAPI {
        constructor() {
            this.baseUrl = CONFIG.apiBaseUrl;
        }

        async request(endpoint, options = {}) {
            const response = await fetch(`${this.baseUrl}${endpoint}`, {
                ...options,
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                credentials: 'include'
            });

            if (response.status === 401) {
                state.isAuthenticated = false;
                throw new Error('Unauthorized');
            }

            return response;
        }

        async syncDiagram(diagramJSON) {
            console.log('[Sync Toolbar] API: Sending sync request...');
            const response = await this.request('/diagrams/sync', {
                method: 'POST',
                body: diagramJSON
            });
            console.log('[Sync Toolbar] API: Response status:', response.status);
            if (!response.ok) {
                const errorText = await response.text();
                console.error('[Sync Toolbar] API: Error response:', errorText);
                // If unauthorized, show helpful message
                if (response.status === 401) {
                    throw new Error('Please login to sync dashboard first');
                }
                throw new Error('Failed to sync: ' + errorText);
            }
            return response.json();
        }
    }

    const api = new SyncAPI();

    // Get current diagram ID from URL
    function getCurrentDiagramId() {
        const match = window.location.pathname.match(/\/diagrams\/([^/]+)/);
        return match ? match[1] : null;
    }

    // ============================================
    // COPIED FROM ChartDB: export-import-utils.ts
    // ============================================

    /**
     * Creates a running ID generator for diagram cloning
     * Copied from: runningIdGenerator() in chartdb/src/lib/export-import-utils.ts
     */
    function createRunningIdGenerator() {
        let id = 0;
        return () => (id++).toString();
    }

    /**
     * Clones a diagram with new IDs for export
     * Copied from: cloneDiagram() in chartdb/src/lib/domain/clone.ts
     * 
     * NOTE: For SYNC operations, we preserve the original diagram ID
     * to avoid creating duplicate diagrams. Only internal entity IDs are regenerated.
     * All references (relationships, dependencies) are properly remapped to new IDs.
     */
    function cloneDiagramForExport(diagram) {
        const generateId = createRunningIdGenerator();
        const idMap = new Map();

        // For SYNC: Preserve the original diagram ID (don't generate new one)
        // This prevents creating duplicate diagrams in IndexedDB
        const clonedDiagram = {
            ...diagram,
            id: diagram.id, // PRESERVE original ChartDB ID
            createdAt: Date.now(),
            updatedAt: Date.now(),
        };

        idMap.set(diagram.id, clonedDiagram.id);

        // Clone tables with new IDs
        if (diagram.tables) {
            clonedDiagram.tables = diagram.tables.map(table => {
                const newTableId = generateId();
                idMap.set(table.id, newTableId);

                const clonedTable = {
                    ...table,
                    id: newTableId,
                    createdAt: table.createdAt || Date.now(),
                };

                // Clone fields with new IDs
                if (table.fields) {
                    clonedTable.fields = table.fields.map(field => {
                        const newFieldId = generateId();
                        idMap.set(field.id, newFieldId);
                        return {
                            ...field,
                            id: newFieldId,
                            createdAt: field.createdAt || Date.now(),
                        };
                    });
                }

                // Update index field references
                if (table.indexes) {
                    clonedTable.indexes = table.indexes.map(index => ({
                        ...index,
                        id: generateId(),
                        fieldIds: index.fieldIds?.map(oldId => idMap.get(oldId) || oldId) || [],
                        createdAt: index.createdAt || Date.now(),
                    }));
                }

                return clonedTable;
            });
        }

        // Clone relationships with updated references - CRITICAL for sync to work
        if (diagram.relationships) {
            clonedDiagram.relationships = diagram.relationships.map(rel => {
                const newSourceTableId = idMap.get(rel.sourceTableId);
                const newTargetTableId = idMap.get(rel.targetTableId);
                const newSourceFieldId = idMap.get(rel.sourceFieldId);
                const newTargetFieldId = idMap.get(rel.targetFieldId);
                
                // Only include relationships where all references can be remapped
                if (!newSourceTableId || !newTargetTableId || !newSourceFieldId || !newTargetFieldId) {
                    console.warn('[cloneDiagramForExport] Skipping relationship with missing references:', rel.id);
                    return null;
                }
                
                return {
                    ...rel,
                    id: generateId(),
                    sourceTableId: newSourceTableId,
                    targetTableId: newTargetTableId,
                    sourceFieldId: newSourceFieldId,
                    targetFieldId: newTargetFieldId,
                    createdAt: rel.createdAt || Date.now(),
                };
            }).filter(rel => rel !== null);
        }

        // Clone dependencies with updated references
        if (diagram.dependencies) {
            clonedDiagram.dependencies = diagram.dependencies.map(dep => {
                const newTableId = idMap.get(dep.tableId);
                const newDependentTableId = idMap.get(dep.dependentTableId);
                
                if (!newTableId || !newDependentTableId) {
                    console.warn('[cloneDiagramForExport] Skipping dependency with missing references:', dep.id);
                    return null;
                }
                
                return {
                    ...dep,
                    id: generateId(),
                    tableId: newTableId,
                    dependentTableId: newDependentTableId,
                    createdAt: dep.createdAt || Date.now(),
                };
            }).filter(dep => dep !== null);
        }

        // Clone areas with new IDs
        if (diagram.areas) {
            clonedDiagram.areas = diagram.areas.map(area => ({
                ...area,
                id: generateId(),
            }));
        }

        // Clone notes with new IDs
        if (diagram.notes) {
            clonedDiagram.notes = diagram.notes.map(note => ({
                ...note,
                id: generateId(),
            }));
        }

        // Clone custom types with new IDs
        if (diagram.customTypes) {
            clonedDiagram.customTypes = diagram.customTypes.map(ct => ({
                ...ct,
                id: generateId(),
            }));
        }

        return clonedDiagram;
    }

    /**
     * Converts diagram to JSON string for export/sync
     * Copied from: diagramToJSONOutput() in chartdb/src/lib/export-import-utils.ts
     * 
     * IMPORTANT: Preserves the original diagram ID to prevent duplicates
     * during sync operations. Only internal entity IDs are regenerated.
     */
    function diagramToJSON(diagram) {
        const clonedDiagram = cloneDiagramForExport(diagram);
        return JSON.stringify(clonedDiagram, null, 2);
    }

    /**
     * Get diagram data in ChartDB JSON export format
     * Uses the duplicated ChartDB export logic above
     */
    async function getCurrentDiagramData() {
        const diagramId = getCurrentDiagramId();
        console.log('[Sync Toolbar] Getting diagram data for ID:', diagramId);
        
        if (!diagramId) {
            console.log('[Sync Toolbar] No diagram ID in URL');
            return null;
        }

        try {
            console.log('[Sync Toolbar] Fetching from IndexedDB...');
            const diagram = await getDiagramFromIndexedDB(diagramId);
            
            if (!diagram) {
                console.log('[Sync Toolbar] Diagram not found in IndexedDB');
                return null;
            }

            console.log('[Sync Toolbar] Diagram found:', diagram.name, 'Tables:', diagram.tables?.length);
            
            // Use copied ChartDB export logic
            const json = diagramToJSON(diagram);
            console.log('[Sync Toolbar] Converted to JSON, length:', json.length);
            return json;
        } catch (e) {
            console.error('[Sync Toolbar] Failed to get diagram:', e);
            return null;
        }
    }

    // Get diagram directly from IndexedDB (read-only, doesn't create)
    async function getDiagramFromIndexedDB(diagramId) {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open('ChartDB');
            
            request.onerror = () => {
                // Database doesn't exist - ChartDB hasn't been opened yet
                resolve(null);
            };
            
            request.onsuccess = () => {
                const db = request.result;
                
                // Check if required stores exist
                if (!db.objectStoreNames.contains('diagrams')) {
                    console.warn('[Sync Toolbar] Database exists but diagrams store not found');
                    db.close();
                    resolve(null);
                    return;
                }
                
                try {
                    // Get diagram metadata
                    const tx = db.transaction(['diagrams'], 'readonly');
                    const store = tx.objectStore('diagrams');
                    const diagramReq = store.get(diagramId);
                    
                    diagramReq.onsuccess = async () => {
                        const diagram = diagramReq.result;
                        if (!diagram) {
                            db.close();
                            resolve(null);
                            return;
                        }

                        // Get all related data
                        const stores = ['db_tables', 'db_relationships', 'db_dependencies', 
                                       'areas', 'notes', 'db_custom_types'];
                        const data = { ...diagram };

                        for (const storeName of stores) {
                            if (!db.objectStoreNames.contains(storeName)) continue;
                            
                            try {
                                const tx2 = db.transaction(storeName, 'readonly');
                                const store2 = tx2.objectStore(storeName);
                                
                                // Try to use index
                                if (store2.indexNames.contains('diagramId')) {
                                    const idx = store2.index('diagramId');
                                    const items = await new Promise((res, rej) => {
                                        const req = idx.getAll(diagramId);
                                        req.onsuccess = () => res(req.result);
                                        req.onerror = () => rej(req.error);
                                    });
                                    
                                    // Map store name to data property
                                    const propName = storeName === 'db_tables' ? 'tables' :
                                                   storeName === 'db_relationships' ? 'relationships' :
                                                   storeName === 'db_dependencies' ? 'dependencies' :
                                                   storeName === 'areas' ? 'areas' :
                                                   storeName === 'notes' ? 'notes' :
                                                   storeName === 'db_custom_types' ? 'customTypes' : storeName;
                                    data[propName] = items;
                                }
                            } catch (e) {
                                console.warn(`Failed to get ${storeName}:`, e);
                            }
                        }
                        
                        db.close();
                        resolve(data);
                    };
                    
                    diagramReq.onerror = () => {
                        db.close();
                        reject(diagramReq.error);
                    };
                } catch (e) {
                    db.close();
                    reject(e);
                }
            };
        });
    }

    // Update current diagram info
    async function updateCurrentDiagramInfo() {
        const diagramId = getCurrentDiagramId();
        state.currentDiagramId = diagramId;
        
        if (diagramId) {
            try {
                const diagram = await getDiagramFromIndexedDB(diagramId);
                state.currentDiagramName = diagram?.name || 'Untitled';
            } catch {
                state.currentDiagramName = 'Untitled';
            }
        } else {
            state.currentDiagramName = null;
        }
    }

    // Check authentication on page load via API
    async function checkAuth() {
        try {
            const response = await fetch('/sync/api/auth/me', {
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            });
            
            if (!response.ok) {
                // User is not authenticated
                state.isAuthenticated = false;
                console.log('[Sync Toolbar] User not authenticated');
                return false;
            }
            state.isAuthenticated = true;
            console.log('[Sync Toolbar] User authenticated');
            return true;
        } catch (err) {
            state.isAuthenticated = false;
            console.error('[Sync Toolbar] Auth check failed:', err);
            return false;
        }
    }

    // Sync current diagram
    async function syncCurrentDiagram() {
        console.log('[Sync Toolbar] Attempting sync...', {
            isAuthenticated: state.isAuthenticated,
            currentDiagramId: state.currentDiagramId,
            syncStatus: state.syncStatus
        });

        // Check auth before syncing
        if (!await checkAuth()) {
            console.log('[Sync Toolbar] Not authenticated, skipping sync');
            state.syncStatus = 'error';
            updateToolbar();
            
            // Redirect to login after 2 seconds
            setTimeout(() => {
                window.location.href = '/sync/login';
            }, 2000);
            return;
        }
        
        if (!state.currentDiagramId) {
            console.log('[Sync Toolbar] No diagram ID, skipping sync');
            return;
        }
        
        if (state.syncStatus === 'syncing') {
            console.log('[Sync Toolbar] Already syncing, skipping');
            return;
        }

        state.syncStatus = 'syncing';
        updateToolbar();

        try {
            console.log('[Sync Toolbar] Getting diagram data...');
            const diagramJSON = await getCurrentDiagramData();
            
            if (!diagramJSON) {
                console.log('[Sync Toolbar] No diagram data found');
                state.syncStatus = 'idle';
                updateToolbar();
                return;
            }

            console.log('[Sync Toolbar] Sending to server...');
            const result = await api.syncDiagram(diagramJSON);
            console.log('[Sync Toolbar] Sync successful:', result);

            state.syncStatus = 'synced';
            state.lastSyncTime = new Date();
            
            // Reset to idle after 3 seconds
            setTimeout(() => {
                if (state.syncStatus === 'synced') {
                    state.syncStatus = 'idle';
                    updateToolbar();
                }
            }, 3000);

        } catch (error) {
            console.error('[Sync Toolbar] Sync failed:', error);
            state.syncStatus = 'error';
            
            // Reset to idle after 5 seconds
            setTimeout(() => {
                if (state.syncStatus === 'error') {
                    state.syncStatus = 'idle';
                    updateToolbar();
                }
            }, 5000);
        }

        updateToolbar();
    }

    // Debounced sync
    function debouncedSync() {
        if (!state.autoSyncEnabled) return;
        
        if (state.debounceTimer) {
            clearTimeout(state.debounceTimer);
        }
        
        state.syncStatus = 'pending';
        updateToolbar();
        
        state.debounceTimer = setTimeout(() => {
            syncCurrentDiagram();
        }, CONFIG.debounceDelay);
    }

    // Monitor IndexedDB for changes
    function setupChangeMonitor() {
        let lastHash = null;
        
        async function checkForChanges() {
            if (!state.autoSyncEnabled || !state.currentDiagramId) {
                return;
            }
            
            try {
                const diagram = await getDiagramFromIndexedDB(state.currentDiagramId);
                if (diagram && diagram.updatedAt) {
                    const updatedAt = new Date(diagram.updatedAt).getTime();
                    if (lastHash !== null && updatedAt > lastHash) {
                        debouncedSync();
                    }
                    lastHash = updatedAt;
                }
            } catch (e) {
                // Ignore errors
            }
        }
        
        // Poll for changes every 3 seconds
        setInterval(checkForChanges, 3000);
    }

    // Load saved preferences
    function loadPreferences() {
        const saved = localStorage.getItem(CONFIG.storageKeys.autoSync);
        state.autoSyncEnabled = saved === null ? true : saved === 'true';
    }

    // Save preferences
    function savePreferences() {
        localStorage.setItem(CONFIG.storageKeys.autoSync, state.autoSyncEnabled);
    }


    // Toggle auto-sync
    function toggleAutoSync() {
        state.autoSyncEnabled = !state.autoSyncEnabled;
        savePreferences();
        updateToolbar();
        
        if (state.autoSyncEnabled) {
            debouncedSync();
        }
    }

    // Create toolbar HTML
    function createToolbar() {
        const toolbar = document.createElement('div');
        toolbar.id = 'chartdb-sync-toolbar';
        toolbar.className = 'chartdb-sync-toolbar';
        return toolbar;
    }

    // Update toolbar UI
    function updateToolbar() {
        const toolbar = document.getElementById('chartdb-sync-toolbar');
        if (!toolbar) return;

        const statusIcons = {
            idle: `<svg class="sync-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/></svg>`,
            pending: `<svg class="sync-icon spinning" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/></svg>`,
            syncing: `<svg class="sync-icon spinning" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/></svg>`,
            synced: `<svg class="sync-icon synced" viewBox="0 0 24 24" fill="currentColor"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>`,
            error: `<svg class="sync-icon error" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>`
        };

        const statusText = {
            idle: 'Cloud Sync',
            pending: 'Pending...',
            syncing: 'Syncing...',
            synced: 'Synced!',
            error: 'Sync Error'
        };

        const diagramInfo = state.currentDiagramName 
            ? `<span class="sync-diagram-name" title="${state.currentDiagramName}">${state.currentDiagramName}</span>` 
            : '';
        
        const lastSyncInfo = state.lastSyncTime 
            ? `<span class="sync-last-time" title="Last synced: ${state.lastSyncTime.toLocaleTimeString()}">· ${formatTimeAgo(state.lastSyncTime)}</span>` 
            : '';

        toolbar.innerHTML = `
            <div class="sync-container">
                <button class="sync-toggle-btn ${state.autoSyncEnabled ? 'enabled' : 'disabled'}" 
                        onclick="window.__chartdbSync.toggleAutoSync()" 
                        title="${state.autoSyncEnabled ? 'Auto-sync ON (click to disable)' : 'Auto-sync OFF (click to enable)'}">
                    <span class="sync-status-indicator ${state.syncStatus}"></span>
                    ${statusIcons[state.syncStatus]}
                    <span class="sync-status-text">${statusText[state.syncStatus]}</span>
                </button>
                ${diagramInfo}
                ${lastSyncInfo}
                <button class="sync-manual-btn" onclick="window.__chartdbSync.syncNow()" title="Sync now">
                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm0 14c-3.31 0-6-2.69-6-6 0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3z"/></svg>
                </button>
                <a href="/sync/dashboard" class="sync-dashboard-btn" target="_blank" title="Open Sync Dashboard">
                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/></svg>
                </a>
            </div>
        `;
    }

    // Format time ago
    function formatTimeAgo(date) {
        const seconds = Math.floor((new Date() - date) / 1000);
        if (seconds < 60) return 'just now';
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
        return date.toLocaleDateString();
    }

    // Find navbar and inject toolbar
    function injectToolbar() {
        const checkNavbar = setInterval(() => {
            const navbar = document.querySelector('nav.flex');
            
            if (navbar) {
                clearInterval(checkNavbar);
                const toolbar = createToolbar();
                const lastChild = navbar.lastElementChild;
                if (lastChild) {
                    navbar.insertBefore(toolbar, lastChild);
                } else {
                    navbar.appendChild(toolbar);
                }
                updateToolbar();
                console.log('ChartDB Sync Toolbar injected into navbar');
            }
        }, 500);

        setTimeout(() => clearInterval(checkNavbar), 30000);
    }

    // Initialize
    async function init() {
        console.log('[Sync Toolbar] Initializing...');
        
        // Load CSS
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = '/static/sync-toolbar.css';
        document.head.appendChild(link);

        // Load preferences
        loadPreferences();
        console.log('[Sync Toolbar] Auto-sync enabled:', state.autoSyncEnabled);

        // Check authentication
        await checkAuth();
        console.log('[Sync Toolbar] Auth status:', state.isAuthenticated);

        // Get current diagram info
        await updateCurrentDiagramInfo();
        console.log('[Sync Toolbar] Current diagram ID:', state.currentDiagramId);

        // Inject toolbar
        injectToolbar();

        // Setup change monitor
        setupChangeMonitor();

        // Expose global functions
        window.__chartdbSync = {
            toggleAutoSync: toggleAutoSync,
            syncNow: syncCurrentDiagram
        };

        // Monitor URL changes
        let lastUrl = window.location.href;
        const urlObserver = new MutationObserver(async () => {
            if (window.location.href !== lastUrl) {
                lastUrl = window.location.href;
                await updateCurrentDiagramInfo();
                updateToolbar();
            }
        });
        urlObserver.observe(document.body, { childList: true, subtree: true });

        window.addEventListener('popstate', async () => {
            await updateCurrentDiagramInfo();
            updateToolbar();
        });

        console.log('[Sync Toolbar] Initialization complete');
    }

    // Delayed init
    function delayedInit() {
        setTimeout(init, 2000);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', delayedInit);
    } else {
        delayedInit();
    }
})();
