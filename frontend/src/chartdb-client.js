// ChartDB IndexedDB Client
// Directly interacts with ChartDB's Dexie.js IndexedDB database

// ChartDB uses 'ChartDB' as the database name
const DB_NAME = 'ChartDB'
// Use version 130 to match ChartDB's current version
const DB_VERSION = 130

class ChartDBClient {
  constructor() {
    this.db = null
  }

  async open() {
    return new Promise((resolve, reject) => {
      // Open with version to handle creation if needed
      const request = indexedDB.open(DB_NAME, DB_VERSION)
      
      request.onerror = () => reject(request.error)
      request.onsuccess = () => {
        this.db = request.result
        resolve(this.db)
      }
      
      // Handle version upgrade - create stores if database was deleted
      request.onupgradeneeded = (event) => {
        console.log('Creating/upgrading ChartDB database structure')
        const db = event.target.result
        
        // Create object stores matching ChartDB's structure
        if (!db.objectStoreNames.contains('diagrams')) {
          db.createObjectStore('diagrams', { keyPath: 'id' })
        }
        
        if (!db.objectStoreNames.contains('db_tables')) {
          const tableStore = db.createObjectStore('db_tables', { keyPath: 'id' })
          tableStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('db_relationships')) {
          const relStore = db.createObjectStore('db_relationships', { keyPath: 'id' })
          relStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('db_dependencies')) {
          const depStore = db.createObjectStore('db_dependencies', { keyPath: 'id' })
          depStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('areas')) {
          const areaStore = db.createObjectStore('areas', { keyPath: 'id' })
          areaStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('notes')) {
          const noteStore = db.createObjectStore('notes', { keyPath: 'id' })
          noteStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('db_custom_types')) {
          const typeStore = db.createObjectStore('db_custom_types', { keyPath: 'id' })
          typeStore.createIndex('diagramId', 'diagramId', { unique: false })
        }
        
        if (!db.objectStoreNames.contains('diagram_filters')) {
          db.createObjectStore('diagram_filters', { keyPath: 'id' })
        }
        
        if (!db.objectStoreNames.contains('config')) {
          db.createObjectStore('config', { keyPath: 'id' })
        }
        
        console.log('ChartDB database structure created')
      }
    })
  }
  
  // Force close and reopen the database
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

  // Get all diagrams
  async getDiagrams() {
    await this.ensureOpen()
    return this.getAllFromStore('diagrams')
  }

  // Get a specific diagram with all related data
  async getDiagramFull(diagramId) {
    await this.ensureOpen()
    
    const diagrams = await this.getAllFromStore('diagrams')
    const diagram = diagrams.find(d => d.id === diagramId)
    
    if (!diagram) {
      throw new Error('Diagram not found')
    }

    // Get related data
    const [tables, relationships, dependencies, areas, notes, customTypes, filters] = await Promise.all([
      this.getByIndex('db_tables', 'diagramId', diagramId),
      this.getByIndex('db_relationships', 'diagramId', diagramId),
      this.getByIndex('db_dependencies', 'diagramId', diagramId),
      this.getByIndex('areas', 'diagramId', diagramId),
      this.getByIndex('notes', 'diagramId', diagramId),
      this.getByIndex('db_custom_types', 'diagramId', diagramId),
      this.getByKey('diagram_filters', diagramId)
    ])

    // For tables, we need to get fields and indexes
    const tablesWithDetails = tables.map(table => ({
      id: table.id,
      name: table.name,
      schema: table.schema,
      x: table.x,
      y: table.y,
      width: table.width,
      color: table.color,
      isView: table.isView,
      isMaterializedView: table.isMaterializedView,
      comments: table.comment,
      order: table.order,
      createdAt: table.createdAt,
      fields: table.fields || [],
      indexes: table.indexes || [],
      expanded: table.expanded,
      parentAreaId: table.parentAreaId
    }))

    return {
      id: diagram.id,
      name: diagram.name,
      databaseType: diagram.databaseType,
      databaseEdition: diagram.databaseEdition,
      createdAt: diagram.createdAt,
      updatedAt: diagram.updatedAt,
      tables: tablesWithDetails,
      relationships: relationships.map(r => ({
        id: r.id,
        name: r.name,
        sourceSchema: r.sourceSchema,
        sourceTableId: r.sourceTableId,
        targetSchema: r.targetSchema,
        targetTableId: r.targetTableId,
        sourceFieldId: r.sourceFieldId,
        targetFieldId: r.targetFieldId,
        sourceCardinality: r.sourceCardinality || r.type?.split('_')[0],
        targetCardinality: r.targetCardinality || r.type?.split('_')[1],
        createdAt: r.createdAt
      })),
      dependencies: dependencies.map(d => ({
        id: d.id,
        schema: d.schema,
        tableId: d.tableId,
        dependentSchema: d.dependentSchema,
        dependentTableId: d.dependentTableId,
        createdAt: d.createdAt
      })),
      areas: areas.map(a => ({
        id: a.id,
        name: a.name,
        x: a.x,
        y: a.y,
        width: a.width,
        height: a.height,
        color: a.color,
        order: a.order
      })),
      notes: notes.map(n => ({
        id: n.id,
        content: n.content,
        x: n.x,
        y: n.y,
        width: n.width,
        height: n.height,
        color: n.color,
        order: n.order
      })),
      customTypes: customTypes.map(ct => ({
        id: ct.id,
        schema: ct.schema,
        name: ct.name || ct.type,
        kind: ct.kind,
        values: ct.values,
        fields: ct.fields,
        order: ct.order
      }))
    }
  }

  // Save a diagram and all related data to IndexedDB
  async saveDiagramFull(diagramData) {
    await this.ensureOpen()
    
    return new Promise((resolve, reject) => {
      const tx = this.db.transaction(
        ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 'areas', 'notes', 'db_custom_types', 'diagram_filters'],
        'readwrite'
      )
      
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => reject(tx.error)
      tx.onabort = () => reject(new Error('Transaction aborted'))
      
      try {
        // Prepare diagram record
        // ChartDB expects createdAt as a number (timestamp), not Date
        const createdAt = diagramData.createdAt 
          ? (typeof diagramData.createdAt === 'number' ? diagramData.createdAt : new Date(diagramData.createdAt).getTime())
          : Date.now()
        
        const diagramRecord = {
          id: diagramData.id,
          name: diagramData.name,
          databaseType: diagramData.databaseType,
          databaseEdition: diagramData.databaseEdition,
          createdAt: createdAt,
          updatedAt: Date.now()
        }

        // Save diagram
        const diagramStore = tx.objectStore('diagrams')
        diagramStore.put(diagramRecord)

        // Clear and save tables
        const tableStore = tx.objectStore('db_tables')
        if (tableStore.indexNames.contains('diagramId')) {
          const tableIndex = tableStore.index('diagramId')
          const tableDeleteReq = tableIndex.openCursor(IDBKeyRange.only(diagramData.id))
          tableDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const table of diagramData.tables || []) {
          // Skip tables without id
          if (!table.id) {
            console.warn('Skipping table without id:', table)
            continue
          }
          
          // ChartDB expects createdAt as number (timestamp)
          const tableCreatedAt = table.createdAt
            ? (typeof table.createdAt === 'number' ? table.createdAt : new Date(table.createdAt).getTime())
            : Date.now()
          
          tableStore.put({
            id: table.id,
            diagramId: diagramData.id,
            name: table.name,
            schema: table.schema,
            x: table.x || 0,
            y: table.y || 0,
            width: table.width,
            color: table.color,
            isView: table.isView || false,
            isMaterializedView: table.isMaterializedView,
            comments: table.comments,
            order: table.order,
            createdAt: tableCreatedAt,
            fields: table.fields || [],
            indexes: table.indexes || [],
            expanded: table.expanded,
            parentAreaId: table.parentAreaId
          })
        }

        // Clear and save relationships
        const relStore = tx.objectStore('db_relationships')
        if (relStore.indexNames.contains('diagramId')) {
          const relIndex = relStore.index('diagramId')
          const relDeleteReq = relIndex.openCursor(IDBKeyRange.only(diagramData.id))
          relDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const rel of diagramData.relationships || []) {
          // Skip relationships without id
          if (!rel.id) {
            console.warn('Skipping relationship without id:', rel)
            continue
          }
          
          // ChartDB expects createdAt as number (timestamp)
          const relCreatedAt = rel.createdAt
            ? (typeof rel.createdAt === 'number' ? rel.createdAt : new Date(rel.createdAt).getTime())
            : Date.now()
          
          relStore.put({
            id: rel.id,
            diagramId: diagramData.id,
            name: rel.name,
            sourceSchema: rel.sourceSchema,
            sourceTableId: rel.sourceTableId,
            targetSchema: rel.targetSchema,
            targetTableId: rel.targetTableId,
            sourceFieldId: rel.sourceFieldId,
            targetFieldId: rel.targetFieldId,
            sourceCardinality: rel.sourceCardinality,
            targetCardinality: rel.targetCardinality,
            type: `${rel.sourceCardinality}_${rel.targetCardinality}`,
            createdAt: relCreatedAt
          })
        }

        // Clear and save dependencies
        const depStore = tx.objectStore('db_dependencies')
        if (depStore.indexNames.contains('diagramId')) {
          const depIndex = depStore.index('diagramId')
          const depDeleteReq = depIndex.openCursor(IDBKeyRange.only(diagramData.id))
          depDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const dep of diagramData.dependencies || []) {
          // Skip dependencies without id
          if (!dep.id) {
            console.warn('Skipping dependency without id:', dep)
            continue
          }
          
          // ChartDB expects createdAt as number (timestamp)
          const depCreatedAt = dep.createdAt
            ? (typeof dep.createdAt === 'number' ? dep.createdAt : new Date(dep.createdAt).getTime())
            : Date.now()
          
          depStore.put({
            id: dep.id,
            diagramId: diagramData.id,
            schema: dep.schema,
            tableId: dep.tableId,
            dependentSchema: dep.dependentSchema,
            dependentTableId: dep.dependentTableId,
            createdAt: depCreatedAt
          })
        }

        // Clear and save areas
        const areaStore = tx.objectStore('areas')
        if (areaStore.indexNames.contains('diagramId')) {
          const areaIndex = areaStore.index('diagramId')
          const areaDeleteReq = areaIndex.openCursor(IDBKeyRange.only(diagramData.id))
          areaDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const area of diagramData.areas || []) {
          // Skip areas without id
          if (!area.id) {
            console.warn('Skipping area without id:', area)
            continue
          }
          
          areaStore.put({
            id: area.id,
            diagramId: diagramData.id,
            name: area.name,
            x: area.x || 0,
            y: area.y || 0,
            width: area.width,
            height: area.height,
            color: area.color,
            order: area.order
          })
        }

        // Clear and save notes
        const noteStore = tx.objectStore('notes')
        if (noteStore.indexNames.contains('diagramId')) {
          const noteIndex = noteStore.index('diagramId')
          const noteDeleteReq = noteIndex.openCursor(IDBKeyRange.only(diagramData.id))
          noteDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const note of diagramData.notes || []) {
          // Skip notes without id
          if (!note.id) {
            console.warn('Skipping note without id:', note)
            continue
          }
          
          noteStore.put({
            id: note.id,
            diagramId: diagramData.id,
            content: note.content,
            x: note.x || 0,
            y: note.y || 0,
            width: note.width,
            height: note.height,
            color: note.color,
            order: note.order
          })
        }

        // Clear and save custom types
        const ctStore = tx.objectStore('db_custom_types')
        if (ctStore.indexNames.contains('diagramId')) {
          const ctIndex = ctStore.index('diagramId')
          const ctDeleteReq = ctIndex.openCursor(IDBKeyRange.only(diagramData.id))
          ctDeleteReq.onsuccess = (e) => {
            const cursor = e.target.result
            if (cursor) {
              cursor.delete()
              cursor.continue()
            }
          }
        }
        
        for (const ct of diagramData.customTypes || []) {
          // Skip custom types without id
          if (!ct.id) {
            console.warn('Skipping custom type without id:', ct)
            continue
          }
          
          ctStore.put({
            id: ct.id,
            diagramId: diagramData.id,
            schema: ct.schema,
            type: ct.name,
            kind: ct.kind,
            values: ct.values,
            fields: ct.fields,
            order: ct.order
          })
        }

      } catch (error) {
        reject(error)
      }
    })
  }

  // Delete a diagram and all related data
  async deleteDiagramFull(diagramId) {
    await this.ensureOpen()
    
    const tx = this.db.transaction(
      ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 'areas', 'notes', 'db_custom_types', 'diagram_filters'],
      'readwrite'
    )

    try {
      await this.deleteByKey(tx, 'diagrams', diagramId)
      await this.deleteByIndex(tx, 'db_tables', 'diagramId', diagramId)
      await this.deleteByIndex(tx, 'db_relationships', 'diagramId', diagramId)
      await this.deleteByIndex(tx, 'db_dependencies', 'diagramId', diagramId)
      await this.deleteByIndex(tx, 'areas', 'diagramId', diagramId)
      await this.deleteByIndex(tx, 'notes', 'diagramId', diagramId)
      await this.deleteByIndex(tx, 'db_custom_types', 'diagramId', diagramId)
      await this.deleteByKey(tx, 'diagram_filters', diagramId)
      
      return true
    } catch (error) {
      tx.abort()
      throw error
    }
  }

  // Helper: Get all records from a store
  getAllFromStore(storeName) {
    return new Promise((resolve, reject) => {
      const tx = this.db.transaction(storeName, 'readonly')
      const store = tx.objectStore(storeName)
      const request = store.getAll()
      
      request.onerror = () => reject(request.error)
      request.onsuccess = () => resolve(request.result || [])
    })
  }

  // Helper: Get records by index
  getByIndex(storeName, indexName, value) {
    return new Promise((resolve, reject) => {
      try {
        const tx = this.db.transaction(storeName, 'readonly')
        const store = tx.objectStore(storeName)
        
        // Check if index exists
        if (!store.indexNames.contains(indexName)) {
          // Fall back to getting all and filtering
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
        resolve([]) // Return empty if store doesn't exist
      }
    })
  }

  // Helper: Get a single record by key
  getByKey(storeName, key) {
    return new Promise((resolve, reject) => {
      try {
        const tx = this.db.transaction(storeName, 'readonly')
        const store = tx.objectStore(storeName)
        const request = store.get(key)
        
        request.onerror = () => reject(request.error)
        request.onsuccess = () => resolve(request.result)
      } catch (error) {
        resolve(null)
      }
    })
  }

  // Helper: Put a record in a store (within existing transaction)
  putInStore(tx, storeName, data) {
    return new Promise((resolve, reject) => {
      const store = tx.objectStore(storeName)
      const request = store.put(data)
      
      request.onerror = () => reject(request.error)
      request.onsuccess = () => resolve(request.result)
    })
  }

  // Helper: Delete by key (within existing transaction)
  deleteByKey(tx, storeName, key) {
    return new Promise((resolve, reject) => {
      const store = tx.objectStore(storeName)
      const request = store.delete(key)
      
      request.onerror = () => reject(request.error)
      request.onsuccess = () => resolve()
    })
  }

  // Helper: Delete all records matching an index value (within existing transaction)
  deleteByIndex(tx, storeName, indexName, value) {
    return new Promise((resolve, reject) => {
      try {
        const store = tx.objectStore(storeName)
        
        if (!store.indexNames.contains(indexName)) {
          // No index, use cursor on all records
          const request = store.openCursor()
          request.onerror = () => reject(request.error)
          request.onsuccess = (event) => {
            const cursor = event.target.result
            if (cursor) {
              if (cursor.value[indexName] === value) {
                cursor.delete()
              }
              cursor.continue()
            } else {
              resolve()
            }
          }
          return
        }
        
        const index = store.index(indexName)
        const request = index.openCursor(IDBKeyRange.only(value))
        
        request.onerror = () => reject(request.error)
        request.onsuccess = (event) => {
          const cursor = event.target.result
          if (cursor) {
            cursor.delete()
            cursor.continue()
          } else {
            resolve()
          }
        }
      } catch (error) {
        resolve() // Ignore if store doesn't exist
      }
    })
  }

  // Check if ChartDB database exists
  async checkDatabase() {
    try {
      const databases = await indexedDB.databases()
      console.log('Available IndexedDB databases:', databases.map(db => db.name))
      
      // Check if ChartDB exists
      const found = databases.some(db => db.name === 'ChartDB')
      console.log('ChartDB found:', found)
      return found
    } catch (error) {
      console.log('indexedDB.databases() not supported, trying to open directly')
      // Firefox doesn't support indexedDB.databases()
      // Try to open the database directly
      try {
        await this.open()
        // Check if the database has the expected object stores
        const hasStores = this.db.objectStoreNames.contains('diagrams')
        console.log('Database opened, has diagrams store:', hasStores)
        return hasStores
      } catch (err) {
        console.error('Failed to open database:', err)
        return false
      }
    }
  }
  
  // List all databases (for debugging)
  async listDatabases() {
    try {
      return await indexedDB.databases()
    } catch {
      return []
    }
  }

  // Clear all diagrams and related data from IndexedDB
  async clearAllDiagrams() {
    await this.ensureOpen()
    
    return new Promise((resolve, reject) => {
      const stores = ['diagrams', 'db_tables', 'db_relationships', 'db_dependencies', 'areas', 'notes', 'db_custom_types', 'diagram_filters']
      const tx = this.db.transaction(stores, 'readwrite')
      
      tx.oncomplete = () => {
        console.log('All diagrams cleared from IndexedDB')
        resolve(true)
      }
      tx.onerror = () => reject(tx.error)
      tx.onabort = () => reject(new Error('Transaction aborted'))
      
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
}

export const chartDB = new ChartDBClient()
