/**
 * Diagram Export/Import Utilities
 * Duplicates ChartDB's export-import-utils.ts logic
 * Maintains independence from ChartDB internal structure
 */

/**
 * Generate a new diagram ID
 */
export function generateDiagramId() {
    return 'diagram_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
}

/**
 * Generate a new ID for any entity
 */
export function generateId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2, 5)
}

/**
 * Running ID generator for cloning
 */
export function createRunningIdGenerator() {
    let id = 0
    return () => (id++).toString()
}

/**
 * Clone a diagram with new IDs
 * This mirrors ChartDB's cloneDiagram function
 * @param {Object} diagram - The diagram to clone
 * @param {Object} options - Options for cloning
 * @param {Function} options.generateId - Custom ID generator function
 * @param {boolean} options.preserveDiagramId - If true, keeps the original diagram ID (useful for sync)
 */
export function cloneDiagram(diagram, options = {}) {
    const idGenerator = options.generateId || generateId
    const preserveDiagramId = options.preserveDiagramId || false
    const idMap = new Map()

    // Clone the diagram (preserve ID if syncing, generate new if importing)
    const clonedDiagram = {
        ...diagram,
        id: preserveDiagramId ? diagram.id : generateDiagramId(),
        createdAt: Date.now(),
        updatedAt: Date.now(),
    }

    // Map old diagram ID to new
    idMap.set(diagram.id, clonedDiagram.id)

    // Clone tables with new IDs
    if (diagram.tables) {
        clonedDiagram.tables = diagram.tables.map(table => {
            const newTableId = idGenerator()
            idMap.set(table.id, newTableId)

            const clonedTable = {
                ...table,
                id: newTableId,
                createdAt: table.createdAt || Date.now(),
            }

            // Clone fields with new IDs
            if (table.fields) {
                clonedTable.fields = table.fields.map(field => {
                    const newFieldId = idGenerator()
                    idMap.set(field.id, newFieldId)
                    return {
                        ...field,
                        id: newFieldId,
                        createdAt: field.createdAt || Date.now(),
                    }
                })
            }

            // Clone indexes with updated field references
            if (table.indexes) {
                clonedTable.indexes = table.indexes.map(index => ({
                    ...index,
                    id: idGenerator(),
                    fieldIds: index.fieldIds?.map(oldId => idMap.get(oldId) || oldId) || [],
                    createdAt: index.createdAt || Date.now(),
                }))
            }

            return clonedTable
        })
    }

    // Clone relationships with updated references - CRITICAL for relationships to work
    if (diagram.relationships) {
        clonedDiagram.relationships = diagram.relationships.map(rel => {
            const newSourceTableId = idMap.get(rel.sourceTableId)
            const newTargetTableId = idMap.get(rel.targetTableId)
            const newSourceFieldId = idMap.get(rel.sourceFieldId)
            const newTargetFieldId = idMap.get(rel.targetFieldId)
            
            // Only include relationships where all references can be remapped
            if (!newSourceTableId || !newTargetTableId || !newSourceFieldId || !newTargetFieldId) {
                console.warn('[cloneDiagram] Skipping relationship with missing references:', rel.id)
                return null
            }
            
            return {
                ...rel,
                id: idGenerator(),
                sourceTableId: newSourceTableId,
                targetTableId: newTargetTableId,
                sourceFieldId: newSourceFieldId,
                targetFieldId: newTargetFieldId,
                createdAt: rel.createdAt || Date.now(),
            }
        }).filter(rel => rel !== null)
    }

    // Clone dependencies with updated references
    if (diagram.dependencies) {
        clonedDiagram.dependencies = diagram.dependencies.map(dep => {
            const newTableId = idMap.get(dep.tableId)
            const newDependentTableId = idMap.get(dep.dependentTableId)
            
            if (!newTableId || !newDependentTableId) {
                console.warn('[cloneDiagram] Skipping dependency with missing references:', dep.id)
                return null
            }
            
            return {
                ...dep,
                id: idGenerator(),
                tableId: newTableId,
                dependentTableId: newDependentTableId,
                createdAt: dep.createdAt || Date.now(),
            }
        }).filter(dep => dep !== null)
    }

    // Clone areas with new IDs
    if (diagram.areas) {
        clonedDiagram.areas = diagram.areas.map(area => ({
            ...area,
            id: idGenerator(),
        }))
    }

    // Clone notes with new IDs
    if (diagram.notes) {
        clonedDiagram.notes = diagram.notes.map(note => ({
            ...note,
            id: idGenerator(),
        }))
    }

    // Clone custom types with new IDs
    if (diagram.customTypes) {
        clonedDiagram.customTypes = diagram.customTypes.map(ct => ({
            ...ct,
            id: idGenerator(),
        }))
    }

    return { diagram: clonedDiagram, idMap }
}

/**
 * Convert diagram to JSON output (for export/server storage)
 * Mirrors ChartDB's diagramToJSONOutput
 */
export function diagramToJSON(diagram) {
    const { diagram: clonedDiagram } = cloneDiagram(diagram, {
        generateId: createRunningIdGenerator()
    })
    return JSON.stringify(clonedDiagram, null, 2)
}

/**
 * Parse diagram from JSON input (for import/server retrieval)
 * Mirrors ChartDB's diagramFromJSONInput
 * 
 * For sync operations: preserves diagram ID but clones all entities with new IDs
 * This prevents ID conflicts while maintaining diagram identity
 */
export function diagramFromJSON(jsonString, options = {}) {
    let loadedDiagram

    try {
        loadedDiagram = JSON.parse(jsonString)
    } catch (e) {
        throw new Error('Invalid JSON format')
    }

    // Validate required fields
    if (!loadedDiagram.id || !loadedDiagram.name) {
        throw new Error('Invalid diagram format: missing required fields')
    }

    // Clone with new IDs but preserve diagram ID for sync
    // This matches ChartDB's behavior but keeps the same diagram identity
    const { diagram } = cloneDiagram(loadedDiagram, {
        preserveDiagramId: true,
        ...options
    })

    return diagram
}

/**
 * Prepare diagram data from IndexedDB for server push
 * Takes raw IndexedDB data and formats it for API
 */
export function prepareDiagramForServer(diagramData) {
    // Ensure all arrays exist
    return {
        id: diagramData.id,
        name: diagramData.name || 'Untitled',
        databaseType: diagramData.databaseType || 'generic',
        databaseEdition: diagramData.databaseEdition,
        tables: diagramData.tables || [],
        relationships: diagramData.relationships || [],
        dependencies: diagramData.dependencies || [],
        areas: diagramData.areas || [],
        notes: diagramData.notes || [],
        customTypes: diagramData.customTypes || [],
        createdAt: diagramData.createdAt,
        updatedAt: diagramData.updatedAt || Date.now(),
    }
}

/**
 * Prepare diagram data from server for IndexedDB storage
 */
export function prepareDiagramForIndexedDB(diagramData) {
    return {
        id: diagramData.id,
        name: diagramData.name || 'Untitled',
        databaseType: diagramData.databaseType || 'generic',
        databaseEdition: diagramData.databaseEdition,
        tables: diagramData.tables || [],
        relationships: diagramData.relationships || [],
        dependencies: diagramData.dependencies || [],
        areas: diagramData.areas || [],
        notes: diagramData.notes || [],
        customTypes: diagramData.customTypes || [],
        createdAt: diagramData.createdAt || Date.now(),
        updatedAt: diagramData.updatedAt || Date.now(),
    }
}
