/**
 * ChartDB Sync Toolbar
 * Integrated toolbar positioned in the top navbar
 * Syncs only the current diagram on IndexedDB changes
 */

(function() {
    'use strict';

    // Configuration
    const CONFIG = {
        syncDashboardUrl: '/sync/',
        apiBaseUrl: '/sync/api',
        debounceDelay: 2000, // Wait 2 seconds after last change before syncing
        storageKeys: {
            authToken: 'chartdb_sync_token',
            autoSync: 'chartdb_sync_auto'
        }
    };

    // State
    let state = {
        isAuthenticated: false,
        autoSyncEnabled: true, // Default on
        syncStatus: 'idle', // idle, syncing, synced, error
        lastSyncTime: null,
        currentDiagramId: null,
        currentDiagramName: null,
        debounceTimer: null,
        isInitialized: false
    };

    // ChartDB IndexedDB Client
    class ChartDBClient {
        constructor() {
            this.dbName = 'ChartDB';
            this.dbVersion = 130;
        }

        async openDB() {
            return new Promise((resolve, reject) => {
                const request = indexedDB.open(this.dbName, this.dbVersion);
                request.onerror = () => reject(request.error);
                request.onsuccess = () => resolve(request.result);
            });
        }

        async getAllFromStore(storeName) {
            const db = await this.openDB();
            return new Promise((resolve, reject) => {
                try {
                    const tx = db.transaction(storeName, 'readonly');
                    const store = tx.objectStore(storeName);
                    const request = store.getAll();
                    request.onerror = () => reject(request.error);
                    request.onsuccess = () => resolve(request.result);
                } catch (e) {
                    resolve([]);
                }
            });
        }

        async getConfig() {
            try {
                const configs = await this.getAllFromStore('config');
                return configs[0] || null;
            } catch {
                return null;
            }
        }

        async getDiagram(diagramId) {
            const diagrams = await this.getAllFromStore('diagrams');
            return diagrams.find(d => d.id === diagramId);
        }

        async getFullDiagram(diagramId) {
            const db = await this.openDB();
            const stores = ['diagrams', 'db_tables', 'db_fields', 'db_indexes', 
                           'db_relationships', 'db_dependencies', 'db_areas', 
                           'diagram_notes', 'db_custom_types'];
            
            const data = {};
            
            for (const storeName of stores) {
                try {
                    const tx = db.transaction(storeName, 'readonly');
                    const store = tx.objectStore(storeName);
                    const allItems = await new Promise((resolve, reject) => {
                        const request = store.getAll();
                        request.onerror = () => reject(request.error);
                        request.onsuccess = () => resolve(request.result);
                    });
                    
                    if (storeName === 'diagrams') {
                        data.diagram = allItems.find(d => d.id === diagramId);
                    } else {
                        data[storeName] = allItems.filter(item => item.diagramId === diagramId);
                    }
                } catch (e) {
                    if (storeName === 'diagrams') {
                        data.diagram = null;
                    } else {
                        data[storeName] = [];
                    }
                }
            }
            
            return data;
        }
    }

    // API Client
    class SyncAPI {
        constructor() {
            this.baseUrl = CONFIG.apiBaseUrl;
        }

        getToken() {
            return localStorage.getItem(CONFIG.storageKeys.authToken);
        }

        async request(endpoint, options = {}) {
            const token = this.getToken();
            const headers = {
                'Content-Type': 'application/json',
                ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
                ...options.headers
            };

            const response = await fetch(`${this.baseUrl}${endpoint}`, {
                ...options,
                headers
            });

            if (response.status === 401) {
                state.isAuthenticated = false;
                updateToolbar();
                throw new Error('Unauthorized');
            }

            return response;
        }

        async checkAuth() {
            try {
                const response = await this.request('/auth/me');
                return response.ok;
            } catch {
                return false;
            }
        }

        async syncDiagram(diagramData) {
            const response = await this.request('/diagrams/sync', {
                method: 'POST',
                body: JSON.stringify(diagramData)
            });
            if (!response.ok) throw new Error('Failed to sync diagram');
            return response.json();
        }
    }

    const chartDB = new ChartDBClient();
    const api = new SyncAPI();

    // Get current diagram ID from URL or config
    async function getCurrentDiagramId() {
        // First try URL: /diagrams/:id
        const match = window.location.pathname.match(/\/diagrams\/([^/]+)/);
        if (match) {
            return match[1];
        }
        
        // Fallback to config's defaultDiagramId
        const config = await chartDB.getConfig();
        return config?.defaultDiagramId || null;
    }

    // Update current diagram info
    async function updateCurrentDiagramInfo() {
        const diagramId = await getCurrentDiagramId();
        state.currentDiagramId = diagramId;
        
        if (diagramId) {
            const diagram = await chartDB.getDiagram(diagramId);
            state.currentDiagramName = diagram?.name || 'Untitled';
        } else {
            state.currentDiagramName = null;
        }
    }

    // Sync current diagram
    async function syncCurrentDiagram() {
        if (!state.isAuthenticated || !state.currentDiagramId || state.syncStatus === 'syncing') {
            return;
        }

        state.syncStatus = 'syncing';
        updateToolbar();

        try {
            const fullDiagram = await chartDB.getFullDiagram(state.currentDiagramId);
            
            if (!fullDiagram.diagram) {
                state.syncStatus = 'idle';
                updateToolbar();
                return;
            }

            await api.syncDiagram({
                id: fullDiagram.diagram.id,
                name: fullDiagram.diagram.name,
                databaseType: fullDiagram.diagram.databaseType,
                databaseEdition: fullDiagram.diagram.databaseEdition,
                tables: fullDiagram.db_tables || [],
                relationships: fullDiagram.db_relationships || [],
                dependencies: fullDiagram.db_dependencies || [],
                areas: fullDiagram.db_areas || [],
                notes: fullDiagram.diagram_notes || [],
                customTypes: fullDiagram.db_custom_types || []
            });

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
            console.error('Sync failed:', error);
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

    // Debounced sync - waits for changes to stop before syncing
    function debouncedSync() {
        if (!state.autoSyncEnabled || !state.isAuthenticated) return;
        
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
        const originalOpen = indexedDB.open.bind(indexedDB);
        
        indexedDB.open = function(name, version) {
            const request = originalOpen(name, version);
            
            if (name === 'ChartDB') {
                request.onsuccess = function(event) {
                    const db = event.target.result;
                    
                    // Intercept transactions to detect writes
                    const originalTransaction = db.transaction.bind(db);
                    db.transaction = function(storeNames, mode) {
                        const tx = originalTransaction(storeNames, mode);
                        
                        if (mode === 'readwrite') {
                            tx.oncomplete = function() {
                                // Diagram data changed, trigger debounced sync
                                debouncedSync();
                            };
                        }
                        
                        return tx;
                    };
                };
            }
            
            return request;
        };
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

    // Check authentication
    async function checkAuth() {
        const token = api.getToken();
        if (!token) {
            state.isAuthenticated = false;
            return;
        }

        try {
            state.isAuthenticated = await api.checkAuth();
        } catch {
            state.isAuthenticated = false;
        }
    }

    // Toggle auto-sync
    function toggleAutoSync() {
        state.autoSyncEnabled = !state.autoSyncEnabled;
        savePreferences();
        updateToolbar();
        
        if (state.autoSyncEnabled) {
            // Sync immediately when enabled
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

        if (!state.isAuthenticated) {
            toolbar.innerHTML = `
                <a href="${CONFIG.syncDashboardUrl}login" class="sync-login-link" target="_blank" title="Sign in to sync">
                    <svg class="sync-icon" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                    </svg>
                    <span>Sign in to sync</span>
                </a>
            `;
        } else {
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
                    <a href="${CONFIG.syncDashboardUrl}dashboard" class="sync-dashboard-btn" target="_blank" title="Open Sync Dashboard">
                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/></svg>
                    </a>
                </div>
            `;
        }
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
        // Wait for ChartDB's navbar to load
        const checkNavbar = setInterval(() => {
            // Look for the top navbar - it has the "Last saved" text
            const navbar = document.querySelector('nav.flex');
            
            if (navbar) {
                clearInterval(checkNavbar);
                
                // Find where to insert - look for the right side of navbar
                const toolbar = createToolbar();
                
                // Insert before the last child or at the end
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

        // Stop trying after 30 seconds
        setTimeout(() => clearInterval(checkNavbar), 30000);
    }

    // Initialize
    async function init() {
        // Load CSS
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = '/static/sync-toolbar.css';
        document.head.appendChild(link);

        // Load preferences
        loadPreferences();

        // Check auth
        await checkAuth();

        // Get current diagram info
        await updateCurrentDiagramInfo();

        // Inject toolbar into navbar
        injectToolbar();

        // Setup change monitor for IndexedDB
        setupChangeMonitor();

        // Expose global functions for onclick handlers
        window.__chartdbSync = {
            toggleAutoSync: toggleAutoSync,
            syncNow: syncCurrentDiagram
        };

        // Monitor URL changes for diagram switches
        let lastUrl = window.location.href;
        const urlObserver = new MutationObserver(async () => {
            if (window.location.href !== lastUrl) {
                lastUrl = window.location.href;
                await updateCurrentDiagramInfo();
                updateToolbar();
            }
        });
        urlObserver.observe(document.body, { childList: true, subtree: true });

        // Also check with popstate
        window.addEventListener('popstate', async () => {
            await updateCurrentDiagramInfo();
            updateToolbar();
        });

        state.isInitialized = true;
        console.log('ChartDB Sync Toolbar initialized');
    }

    // Wait for DOM
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
