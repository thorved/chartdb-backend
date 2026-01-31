// ChartDB IndexedDB Client - Read/Write Only
// Does NOT create database - ChartDB must be opened first

const DB_NAME = 'ChartDB'

class ChartDBClient {
    constructor() {
        this.db = null
    }

    async open() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(DB_NAME)

            request.onerror = () => {
                reject(new Error('ChartDB database not found. Please open ChartDB app first.'))
            }
            
            request.onsuccess = () => {
                this.db = request.result
                console.log('[ChartDB Client] Database opened, version:', this.db.version)
                console.log('[ChartDB Client] Available stores:', Array.from(this.db.objectStoreNames))
                resolve(this.db)
            }
        })
    }

    async reopen() {
        if (this.db) {
            this.db.close()
            this.db = null
        }
        return this.open()
    }

    async ensureOpen() {
        if (!this.db) {
            await this.open()
        }
        return this.db
    }

    async checkDatabase() {
        try {
            const databases = await indexedDB.databases()
            const found = databases.some(db => db.name === 'ChartDB')
            
            if (!found) {
                return false
            }
            
            try {
                await this.open()
                const hasStores = this.db.objectStoreNames.contains('diagrams') && 
                                 this.db.objectStoreNames.contains('db_tables')
                
                if (!hasStores) {
                    console.warn('[ChartDB Client] Database exists but is empty (no object stores)')
                    this.db.close()
                    this.db = null
                    return false
                }
                
                return true
            } catch (err) {
                return false
            }
        } catch (error) {
            try {
                await this.open()
                const hasStores = this.db.objectStoreNames.contains('diagrams')
                return hasStores
            } catch (err) {
                return false
            }
        }
    }

    async getDiagrams() {
        await this.ensureOpen()
        return this.getAllFromStore('diagrams')
    }

    async getDiagramJSON(diagramId) {
        await this.ensureOpen()
        
        const diagram = await this.getFromStore('diagrams', diagramId)
        
        if (!diagram) {
            throw new Error('Diagram not found')
        }

        const [tables, relationships, dependencies, areas, notes, customTypes] = await Promise.all([
            this.getByIndex('db_tables', 'diagramId', diagramId),
            this.getByIndex('db_relationships', 'diagramId', diagramId),
            this.getByIndex('db_dependencies', 'diagramId', diagramId),
            this.getByIndex('areas', 'diagramId', diagramId),
            this.getByIndex('notes', 'diagramId', diagramId),
            this.getByIndex('db_custom_types', 'diagramId', diagramId),
        ])

        return {
            id: diagram.id,
            name: diagram.name,
            databaseType: diagram.databaseType,
            databaseEdition: diagram.databaseEdition,
            createdAt: diagram.createdAt,
            updatedAt: diagram.updatedAt,
            tables: tables || [],
            relationships: relationships || [],
            dependencies: dependencies || [],
            areas: areas || [],
            notes: notes || [],
            customTypes: customTypes || [],
        }
    }

    async saveDiagramJSON(diagramData) {
        await this.ensureOpen()
        
        console.log('[ChartDB Client] Saving diagram:', diagramData.id, diagramData.name)
        console.log('[ChartDB Client] Tables count:', diagramData.tables?.length || 0)
        console.log('[ChartDB Client] Relationships count:', diagramData.relationships?.length || 0)

        return new Promise((resolve, reject) => {
            // Check all required stores exist
            const requiredStores = ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 
                                   'areas', 'notes', 'db_custom_types']
            const missingStores = requiredStores.filter(store => !this.db.objectStoreNames.contains(store))
            
            if (missingStores.length > 0) {
                console.error('[ChartDB Client] Missing object stores:', missingStores)
                reject(new Error(`Missing object stores: ${missingStores.join(', ')}`))
                return
            }

            const tx = this.db.transaction(
                ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 
                 'areas', 'notes', 'db_custom_types'],
                'readwrite'
            )

            tx.oncomplete = () => {
                console.log('[ChartDB Client] Diagram saved successfully:', diagramData.id)
                resolve(true)
            }
            tx.onerror = () => {
                console.error('[ChartDB Client] Transaction error:', tx.error)
                reject(tx.error)
            }

            try {
                // Save diagram metadata
                console.log('[ChartDB Client] Saving diagram metadata...')
                const diagramStore = tx.objectStore('diagrams')
                diagramStore.put({
                    id: diagramData.id,
                    name: diagramData.name,
                    databaseType: diagramData.databaseType,
                    databaseEdition: diagramData.databaseEdition,
                    // ChartDB expects Date objects, not timestamps
                    createdAt: diagramData.createdAt ? new Date(diagramData.createdAt) : new Date(),
                    updatedAt: diagramData.updatedAt ? new Date(diagramData.updatedAt) : new Date(),
                })

                // Clear existing data
                console.log('[ChartDB Client] Clearing existing data...')
                this.clearByIndex(tx, 'db_tables', 'diagramId', diagramData.id)
                this.clearByIndex(tx, 'db_relationships', 'diagramId', diagramData.id)
                this.clearByIndex(tx, 'db_dependencies', 'diagramId', diagramData.id)
                this.clearByIndex(tx, 'areas', 'diagramId', diagramData.id)
                this.clearByIndex(tx, 'notes', 'diagramId', diagramData.id)
                this.clearByIndex(tx, 'db_custom_types', 'diagramId', diagramData.id)

                // Save tables with fields/indexes inline (ChartDB format)
                if (diagramData.tables && diagramData.tables.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.tables.length, 'tables...')
                    const tablesStore = tx.objectStore('db_tables')
                    diagramData.tables.forEach(table => {
                        // IMPORTANT: ChartDB stores fields and indexes INLINE in the table object
                        // Convert timestamps to Date objects for ChartDB compatibility
                        const tableData = {
                            ...table,
                            diagramId: diagramData.id,
                        }
                        // Convert date fields if they exist
                        if (tableData.createdAt && typeof tableData.createdAt === 'number') {
                            tableData.createdAt = new Date(tableData.createdAt)
                        }
                        tablesStore.put(tableData)
                    })
                }

                // Save relationships
                if (diagramData.relationships && diagramData.relationships.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.relationships.length, 'relationships...')
                    const relStore = tx.objectStore('db_relationships')
                    diagramData.relationships.forEach(relationship => {
                        relStore.put({
                            ...relationship,
                            diagramId: diagramData.id,
                        })
                    })
                }

                // Save dependencies
                if (diagramData.dependencies && diagramData.dependencies.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.dependencies.length, 'dependencies...')
                    const depStore = tx.objectStore('db_dependencies')
                    diagramData.dependencies.forEach(dependency => {
                        depStore.put({
                            ...dependency,
                            diagramId: diagramData.id,
                        })
                    })
                }

                // Save areas
                if (diagramData.areas && diagramData.areas.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.areas.length, 'areas...')
                    const areasStore = tx.objectStore('areas')
                    diagramData.areas.forEach(area => {
                        areasStore.put({
                            ...area,
                            diagramId: diagramData.id,
                        })
                    })
                }

                // Save notes
                if (diagramData.notes && diagramData.notes.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.notes.length, 'notes...')
                    const notesStore = tx.objectStore('notes')
                    diagramData.notes.forEach(note => {
                        notesStore.put({
                            ...note,
                            diagramId: diagramData.id,
                        })
                    })
                }

                // Save custom types
                if (diagramData.customTypes && diagramData.customTypes.length > 0) {
                    console.log('[ChartDB Client] Saving', diagramData.customTypes.length, 'custom types...')
                    const ctStore = tx.objectStore('db_custom_types')
                    diagramData.customTypes.forEach(customType => {
                        ctStore.put({
                            ...customType,
                            diagramId: diagramData.id,
                        })
                    })
                }

            } catch (error) {
                console.error('[ChartDB Client] Save error:', error)
                reject(error)
            }
        })
    }

    async deleteDiagram(diagramId) {
        await this.ensureOpen()
        
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(
                ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 
                 'areas', 'notes', 'db_custom_types'],
                'readwrite'
            )

            tx.oncomplete = () => resolve(true)
            tx.onerror = () => reject(tx.error)

            tx.objectStore('diagrams').delete(diagramId)
            this.clearByIndex(tx, 'db_tables', 'diagramId', diagramId)
            this.clearByIndex(tx, 'db_relationships', 'diagramId', diagramId)
            this.clearByIndex(tx, 'db_dependencies', 'diagramId', diagramId)
            this.clearByIndex(tx, 'areas', 'diagramId', diagramId)
            this.clearByIndex(tx, 'notes', 'diagramId', diagramId)
            this.clearByIndex(tx, 'db_custom_types', 'diagramId', diagramId)
        })
    }

    async clearAllDiagrams() {
        await this.ensureOpen()

        const stores = ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 
                       'areas', 'notes', 'db_custom_types']

        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(stores, 'readwrite')

            tx.oncomplete = () => {
                console.log('[ChartDB Client] All diagrams cleared')
                resolve(true)
            }
            tx.onerror = () => reject(tx.error)

            try {
                for (const storeName of stores) {
                    const store = tx.objectStore(storeName)
                    store.clear()
                }
            } catch (error) {
                reject(error)
            }
        })
    }

    // Helper methods
    getFromStore(storeName, id) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(storeName, 'readonly')
            const store = tx.objectStore(storeName)
            const request = store.get(id)
            request.onerror = () => reject(request.error)
            request.onsuccess = () => resolve(request.result)
        })
    }

    getAllFromStore(storeName) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(storeName, 'readonly')
            const store = tx.objectStore(storeName)
            const request = store.getAll()
            request.onerror = () => reject(request.error)
            request.onsuccess = () => resolve(request.result || [])
        })
    }

    getByIndex(storeName, indexName, value) {
        return new Promise((resolve, reject) => {
            try {
                const tx = this.db.transaction(storeName, 'readonly')
                const store = tx.objectStore(storeName)

                if (!store.indexNames.contains(indexName)) {
                    const request = store.getAll()
                    request.onerror = () => reject(request.error)
                    request.onsuccess = () => {
                        const results = (request.result || []).filter(item => item[indexName] === value)
                        resolve(results)
                    }
                    return
                }

                const index = store.index(indexName)
                const request = index.getAll(value)
                request.onerror = () => reject(request.error)
                request.onsuccess = () => resolve(request.result || [])
            } catch (error) {
                resolve([])
            }
        })
    }

    clearByIndex(tx, storeName, indexName, value) {
        try {
            const store = tx.objectStore(storeName)
            if (store.indexNames.contains(indexName)) {
                const request = store.index(indexName).openCursor(IDBKeyRange.only(value))
                request.onsuccess = (e) => {
                    const cursor = e.target.result
                    if (cursor) {
                        cursor.delete()
                        cursor.continue()
                    }
                }
            }
        } catch (e) {
            // Ignore
        }
    }

    async listDatabases() {
        try {
            return await indexedDB.databases()
        } catch {
            return []
        }
    }
}

export const chartDB = new ChartDBClient()
